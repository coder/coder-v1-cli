package wsnet

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"

	"cdr.dev/coder-cli/coder-sdk"
	"github.com/pion/webrtc/v3"
	"nhooyr.io/websocket"
)

// DialConfig provides options to configure the Dial for a connection.
type DialConfig struct {
	ICEServers []webrtc.ICEServer
}

// Dial connects to the broker and negotiates a connection to a listener.
//
func Dial(ctx context.Context, broker string, config *DialConfig) (*Dialer, error) {
	if config == nil {
		config = &DialConfig{}
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
		ctrl: ctrl,
		rtc:  rtc,
	}

	go func() {
		err = waitForDataChannelOpen(ctx, ctrl)
		if err != nil {
			_ = conn.Close(websocket.StatusAbnormalClosure, "timeout")
			return
		}
		_ = conn.Close(websocket.StatusNormalClosure, "connected")
	}()

	return dialer, dialer.negotiate(nconn)
}

type Dialer struct {
	ctrl *webrtc.DataChannel
	rtc  *webrtc.PeerConnection
}

func (d *Dialer) negotiate(nconn net.Conn) (err error) {
	decoder := json.NewDecoder(nconn)
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
	return nil
}

func (d *Dialer) Close() error {
	return nil
}

func (d *Dialer) Ping(ctx context.Context) error {
	return nil
}

func (d *Dialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return nil, nil
}
