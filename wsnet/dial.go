package wsnet

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/pion/datachannel"
	"github.com/pion/webrtc/v3"
	"golang.org/x/net/proxy"
	"nhooyr.io/websocket"

	"cdr.dev/coder-cli/coder-sdk"
)

// DialOptions are configurable options for a wsnet connection.
type DialOptions struct {
	// ICEServers is an array of STUN or TURN servers to use for negotiation purposes.
	// See: https://developer.mozilla.org/en-US/docs/Web/API/RTCConfiguration/iceServers
	ICEServers []webrtc.ICEServer

	// TURNProxyAuthToken is used to authenticate a TURN proxy request.
	TURNProxyAuthToken string

	// TURNProxyURL is the URL to proxy all TURN data through.
	// This URL is sent to the listener during handshake so both
	// ends connect to the same TURN endpoint.
	TURNProxyURL *url.URL
}

// DialWebsocket dials the broker with a WebSocket and negotiates a connection.
func DialWebsocket(ctx context.Context, broker string, netOpts *DialOptions, wsOpts *websocket.DialOptions) (*Dialer, error) {
	conn, resp, err := websocket.Dial(ctx, broker, wsOpts)
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
	return Dial(ctx, nconn, netOpts)
}

// Dial negotiates a connection to a listener.
func Dial(ctx context.Context, conn net.Conn, options *DialOptions) (*Dialer, error) {
	if options == nil {
		options = &DialOptions{}
	}
	if options.ICEServers == nil {
		options.ICEServers = []webrtc.ICEServer{}
	}

	var turnProxy proxy.Dialer
	if options.TURNProxyURL != nil {
		turnProxy = &turnProxyDialer{
			baseURL: options.TURNProxyURL,
			token:   options.TURNProxyAuthToken,
		}
	}
	rtc, err := newPeerConnection(options.ICEServers, turnProxy)
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

	var turnProxyURL string
	if options.TURNProxyURL != nil {
		turnProxyURL = options.TURNProxyURL.String()
	}

	offerMessage, err := json.Marshal(&BrokerMessage{
		Offer:        &offer,
		Servers:      options.ICEServers,
		TURNProxyURL: turnProxyURL,
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
		connClosers: []io.Closer{ctrl},
	}

	return dialer, dialer.negotiate(ctx)
}

// Dialer enables arbitrary dialing to any network and address
// inside a workspace. The opposing end of the WebSocket messages
// should be proxied with a Listener.
type Dialer struct {
	conn   net.Conn
	ctrl   *webrtc.DataChannel
	ctrlrw datachannel.ReadWriteCloser
	rtc    *webrtc.PeerConnection

	connClosers    []io.Closer
	connClosersMut sync.Mutex
	pingMut        sync.Mutex
}

func (d *Dialer) negotiate(ctx context.Context) (err error) {
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

			// Close connections opened when RTC was alive.
			d.connClosersMut.Lock()
			defer d.connClosersMut.Unlock()
			for _, connCloser := range d.connClosers {
				_ = connCloser.Close()
			}
			d.connClosers = make([]io.Closer, 0)
		})
	}()

	go func() {
		// If a connection is opened but the other end may not be, negotiation
		// can get stuck forever. We don't want this, so we must listen to the
		// context as well.
		<-ctx.Done()
		select {
		case <-errCh:
		default:
			errCh <- ctx.Err()
			close(errCh)
		}
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

// ActiveConnections returns the amount of active connections.
// DialContext opens a connection, and close will end it.
func (d *Dialer) activeConnections() int {
	stats, ok := d.rtc.GetStats().GetConnectionStats(d.rtc)
	if !ok {
		return -1
	}
	// Subtract 1 for the control channel.
	return int(stats.DataChannelsRequested-stats.DataChannelsClosed) - 1
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
	err := waitForDataChannelOpen(ctx, d.ctrl)
	if err != nil {
		return err
	}
	if d.ctrlrw == nil {
		d.ctrlrw, err = d.ctrl.Detach()
		if err != nil {
			return err
		}
	}
	d.pingMut.Lock()
	defer d.pingMut.Unlock()
	_, err = d.ctrlrw.Write([]byte{'a'})
	if err != nil {
		return fmt.Errorf("write: %w", err)
	}
	errCh := make(chan error)
	go func() {
		// There's a race in which connections can get lost-mid ping
		// in which case this would block forever.
		defer close(errCh)
		_, err = d.ctrlrw.Read(make([]byte, 4))
		errCh <- err
	}()
	ctx, cancelFunc := context.WithTimeout(ctx, time.Second*15)
	defer cancelFunc()
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
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

	c := &dataChannelConn{
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
