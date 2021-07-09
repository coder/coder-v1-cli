package wsnet

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/pion/datachannel"
	"github.com/pion/webrtc/v3"
	"nhooyr.io/websocket"

	"cdr.dev/coder-cli/coder-sdk"
)

// DialWebsocket dials the broker with a WebSocket and negotiates a connection.
func DialWebsocket(ctx context.Context, broker string, iceServers []webrtc.ICEServer) (*Dialer, error) {
	conn, resp, err := websocket.Dial(ctx, broker, nil)
	if err != nil {
		if resp != nil {
			defer func() {
				_ = resp.Body.Close()
			}()
			return nil, coder.NewHTTPError(resp)
		}
		return nil, fmt.Errorf("dial websocket: %w", err)
	}
	nconn := websocket.NetConn(ctx, conn, websocket.MessageBinary)
	defer func() {
		_ = nconn.Close()
		// We should close the socket intentionally.
		_ = conn.Close(websocket.StatusInternalError, "an error occurred")
	}()
	return Dial(nconn, iceServers)
}

// Dial negotiates a connection to a listener.
func Dial(conn net.Conn, iceServers []webrtc.ICEServer) (*Dialer, error) {
	if iceServers == nil {
		iceServers = []webrtc.ICEServer{}
	}

	rtc, err := newPeerConnection(iceServers)
	if err != nil {
		return nil, fmt.Errorf("create peer connection: %w", err)
	}

	flushCandidates := proxyICECandidates(rtc, conn)

	ctrl, err := rtc.CreateDataChannel(controlChannel, &webrtc.DataChannelInit{
		Protocol: stringPtr(controlChannel),
		Ordered:  boolPtr(true),
	})
	if err != nil {
		return nil, fmt.Errorf("create control channel: %w", err)
	}

	offer, err := rtc.CreateOffer(&webrtc.OfferOptions{})
	if err != nil {
		return nil, fmt.Errorf("create offer: %w", err)
	}
	err = rtc.SetLocalDescription(offer)
	if err != nil {
		return nil, fmt.Errorf("set local offer: %w", err)
	}

	offerMessage, err := json.Marshal(&BrokerMessage{
		Offer:   &offer,
		Servers: iceServers,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal offer message: %w", err)
	}
	_, err = conn.Write(offerMessage)
	if err != nil {
		return nil, fmt.Errorf("write offer: %w", err)
	}
	flushCandidates()

	dialer := &Dialer{
		conn:        conn,
		ctrl:        ctrl,
		rtc:         rtc,
		closedChan:  make(chan struct{}),
		connClosers: make([]io.Closer, 0),
	}

	return dialer, dialer.negotiate()
}

// Dialer enables arbitrary dialing to any network and address
// inside a workspace. The opposing end of the WebSocket messages
// should be proxied with a Listener.
type Dialer struct {
	conn   net.Conn
	ctrl   *webrtc.DataChannel
	ctrlrw datachannel.ReadWriteCloser
	rtc    *webrtc.PeerConnection

	closedChan     chan struct{}
	connClosers    []io.Closer
	connClosersMut sync.Mutex
}

func (d *Dialer) negotiate() (err error) {
	var (
		decoder = json.NewDecoder(d.conn)
		errCh   = make(chan error)
		// If candidates are sent before an offer, we place them here.
		// We currently have no assurances to ensure this can't happen,
		// so it's better to buffer and process than fail.
		pendingCandidates = []webrtc.ICECandidateInit{}
	)

	go func() {
		defer close(errCh)
		defer func() {
			_ = d.conn.Close()
		}()
		err := waitForConnectionOpen(context.Background(), d.rtc)
		if err != nil {
			errCh <- err
			return
		}
		d.rtc.OnConnectionStateChange(func(pcs webrtc.PeerConnectionState) {
			if pcs == webrtc.PeerConnectionStateConnected {
				return
			}

			// Close connections opened while the RTC was alive.
			d.connClosersMut.Lock()
			defer d.connClosersMut.Unlock()
			for _, connCloser := range d.connClosers {
				_ = connCloser.Close()
			}
			d.connClosers = make([]io.Closer, 0)

			select {
			case <-d.closedChan:
				return
			default:
			}
			close(d.closedChan)
		})
	}()

	for {
		var msg BrokerMessage
		err = decoder.Decode(&msg)
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrClosedPipe) {
			break
		}
		if err != nil {
			return fmt.Errorf("read: %w", err)
		}
		if msg.Candidate != "" {
			c := webrtc.ICECandidateInit{
				Candidate: msg.Candidate,
			}
			if d.rtc.RemoteDescription() == nil {
				pendingCandidates = append(pendingCandidates, c)
				continue
			}
			err = d.rtc.AddICECandidate(c)
			if err != nil {
				return fmt.Errorf("accept ice candidate: %s: %w", msg.Candidate, err)
			}
			continue
		}
		if msg.Answer != nil {
			err = d.rtc.SetRemoteDescription(*msg.Answer)
			if err != nil {
				return fmt.Errorf("set answer: %w", err)
			}
			for _, candidate := range pendingCandidates {
				err = d.rtc.AddICECandidate(candidate)
				if err != nil {
					return fmt.Errorf("accept pending ice candidate: %s: %w", candidate.Candidate, err)
				}
			}
			pendingCandidates = nil
			continue
		}
		if msg.Error != "" {
			return errors.New(msg.Error)
		}
		return fmt.Errorf("unhandled message: %+v", msg)
	}
	return <-errCh
}

// Closed returns a channel that closes when
// the connection is closed.
func (d *Dialer) Closed() <-chan struct{} {
	return d.closedChan
}

// Close closes the RTC connection.
// All data channels dialed will be closed.
func (d *Dialer) Close() error {
	return d.rtc.Close()
}

// Ping sends a ping through the control channel.
func (d *Dialer) Ping(ctx context.Context) error {
	if d.ctrl.ReadyState() == webrtc.DataChannelStateClosed || d.ctrl.ReadyState() == webrtc.DataChannelStateClosing {
		return webrtc.ErrConnectionClosed
	}
	// Since we control the client and server we could open this
	// data channel with `Negotiated` true to reduce traffic being
	// sent when the RTC connection is opened.
	err := waitForDataChannelOpen(context.Background(), d.ctrl)
	if err != nil {
		return err
	}
	if d.ctrlrw == nil {
		d.ctrlrw, err = d.ctrl.Detach()
		if err != nil {
			return err
		}
	}
	_, err = d.ctrlrw.Write([]byte{'a'})
	if err != nil {
		return fmt.Errorf("write: %w", err)
	}
	b := make([]byte, 4)
	_, err = d.ctrlrw.Read(b)
	return err
}

// DialContext dials the network and address on the remote listener.
func (d *Dialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	dc, err := d.rtc.CreateDataChannel("proxy", &webrtc.DataChannelInit{
		Ordered:  boolPtr(network != "udp"),
		Protocol: stringPtr(fmt.Sprintf("%s:%s", network, address)),
	})
	if err != nil {
		return nil, fmt.Errorf("create data channel: %w", err)
	}
	d.connClosersMut.Lock()
	d.connClosers = append(d.connClosers, dc)
	d.connClosersMut.Unlock()

	err = waitForDataChannelOpen(ctx, dc)
	if err != nil {
		return nil, fmt.Errorf("wait for open: %w", err)
	}
	rw, err := dc.Detach()
	if err != nil {
		return nil, fmt.Errorf("detach: %w", err)
	}

	errCh := make(chan error)
	go func() {
		var res DialChannelResponse
		err = json.NewDecoder(rw).Decode(&res)
		if err != nil {
			errCh <- fmt.Errorf("read dial response: %w", err)
			return
		}
		if res.Err == "" {
			close(errCh)
			return
		}
		err := errors.New(res.Err)
		if res.Code == CodeDialErr {
			err = &net.OpError{
				Op:  res.Op,
				Net: res.Net,
				Err: err,
			}
		}
		errCh <- err
	}()
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	select {
	case err := <-errCh:
		if err != nil {
			return nil, err
		}
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	c := &conn{
		addr: &net.UnixAddr{
			Name: address,
			Net:  network,
		},
		dc: dc,
		rw: rw,
	}
	c.init()
	return c, nil
}
