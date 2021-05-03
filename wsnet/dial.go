package wsnet

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/pion/datachannel"
	"github.com/pion/webrtc/v3"
	"nhooyr.io/websocket"

	"cdr.dev/coder-cli/coder-sdk"
)

// DialConfig provides options to configure the Dial for a connection.
type DialConfig struct {
	ICEServers []webrtc.ICEServer
}

// Dial connects to the broker and negotiates a connection to a listener.
func Dial(ctx context.Context, broker string, config *DialConfig) (*Dialer, error) {
	if config == nil {
		config = &DialConfig{
			ICEServers: []webrtc.ICEServer{},
		}
	}

	conn, resp, err := websocket.Dial(ctx, broker, nil)
	if err != nil {
		if resp != nil {
			defer func() {
				_ = resp.Body.Close()
			}()
			return nil, &coder.HTTPError{
				Response: resp,
			}
		}
		return nil, fmt.Errorf("dial websocket: %w", err)
	}
	nconn := websocket.NetConn(ctx, conn, websocket.MessageBinary)
	defer func() {
		_ = nconn.Close()
		// We should close the socket intentionally.
		_ = conn.Close(websocket.StatusInternalError, "an error occurred")
	}()

	rtc, err := newPeerConnection(config.ICEServers)
	if err != nil {
		return nil, fmt.Errorf("create peer connection: %w", err)
	}

	flushCandidates := proxyICECandidates(rtc, nconn)

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

	offerMessage, err := json.Marshal(&protoMessage{
		Offer:   &offer,
		Servers: config.ICEServers,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal offer message: %w", err)
	}
	_, err = nconn.Write(offerMessage)
	if err != nil {
		return nil, fmt.Errorf("write offer: %w", err)
	}
	flushCandidates()

	dialer := &Dialer{
		ws:   conn,
		ctrl: ctrl,
		rtc:  rtc,
	}

	return dialer, dialer.negotiate(nconn)
}

// Dialer enables arbitrary dialing to any network and address
// inside a workspace. The opposing end of the WebSocket messages
// should be proxied with a Listener.
type Dialer struct {
	ws     *websocket.Conn
	ctrl   *webrtc.DataChannel
	ctrlrw datachannel.ReadWriteCloser
	rtc    *webrtc.PeerConnection
}

func (d *Dialer) negotiate(nconn net.Conn) (err error) {
	decoder := json.NewDecoder(nconn)
	errCh := make(chan error)
	go func() {
		defer close(errCh)
		err := waitForDataChannelOpen(context.Background(), d.ctrl)
		if err != nil {
			_ = d.ws.Close(websocket.StatusAbnormalClosure, "timeout")
			errCh <- err
			return
		}
		d.ctrlrw, err = d.ctrl.Detach()
		if err != nil {
			errCh <- err
		}
		_ = d.ws.Close(websocket.StatusNormalClosure, "connected")
	}()

	for {
		var msg protoMessage
		err = decoder.Decode(&msg)
		if errors.Is(err, io.EOF) {
			break
		}
		if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
			// The listener closed the socket because success!
			break
		}
		if err != nil {
			return fmt.Errorf("read: %w", err)
		}
		if msg.Candidate != "" {
			err = d.rtc.AddICECandidate(webrtc.ICECandidateInit{
				Candidate: msg.Candidate,
			})
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
			continue
		}
		if msg.Error != "" {
			return errors.New(msg.Error)
		}
		return fmt.Errorf("unhandled message: %+v", msg)
	}
	return <-errCh
}

// Close closes the RTC connection.
// All data channels dialed will be closed.
func (d *Dialer) Close() error {
	return d.rtc.Close()
}

// Ping sends a ping through the control channel.
func (d *Dialer) Ping(ctx context.Context) error {
	_, err := d.ctrlrw.Write([]byte{'a'})
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
	dc.OnError(func(err error) {
		fmt.Printf("We got err %+v\n", err)
	})
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
		var init dialChannelMessage
		err = json.NewDecoder(rw).Decode(&init)
		if err != nil {
			errCh <- fmt.Errorf("read init: %w", err)
			return
		}
		if init.Err == "" {
			close(errCh)
			return
		}
		err := errors.New(init.Err)
		if init.Net != "" {
			errCh <- &net.OpError{
				Op:  init.Op,
				Net: init.Net,
				Err: err,
			}
			return
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

	return &conn{
		addr: &net.UnixAddr{
			Name: address,
			Net:  network,
		},
		rw: rw,
	}, nil
}
