package wsnet

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"

	"cdr.dev/coder-cli/coder-sdk"
	"github.com/hashicorp/yamux"
	"github.com/pion/webrtc/v3"
	"nhooyr.io/websocket"
)

func Listen(ctx context.Context, broker string) (net.Listener, error) {
	conn, resp, err := websocket.Dial(ctx, broker, nil)
	if err != nil {
		if resp != nil {
			return nil, &coder.HTTPError{
				Response: resp,
			}
		}
		return nil, err
	}
	nconn := websocket.NetConn(ctx, conn, websocket.MessageBinary)
	session, err := yamux.Server(nconn, nil)
	if err != nil {
		return nil, fmt.Errorf("")
	}
	return nil, nil
}

type listener struct {
	session *yamux.Session
}

func (l *listener) Accept() (net.Conn, error) {
	conn, err := l.session.Accept()
	if err != nil {
		return nil, err
	}

	var (
		decoder    = json.NewDecoder(conn)
		closeError = func(err error) error {
			d, _ := json.Marshal(&protoMessage{
				Error: err.Error(),
			})
			_, _ = conn.Write(d)
			_ = conn.Close()
			return err
		}
		rtc *webrtc.PeerConnection
	)

	for {
		var msg protoMessage
		err = decoder.Decode(&msg)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if msg.Candidate != "" {
			if rtc == nil {
				return nil, closeError(fmt.Errorf("Offer must be sent before candidates"))
			}

			err = rtc.AddICECandidate(webrtc.ICECandidateInit{
				Candidate: msg.Candidate,
			})
			if err != nil {
				return nil, closeError(fmt.Errorf("accept ice candidate: %w", err))
			}
		}

		if msg.Offer != nil {
			if msg.Servers == nil {
				return nil, closeError(fmt.Errorf("ICEServers must be provided"))
			}
			rtc, err = newPeerConnection(msg.Servers)
			if err != nil {
				return nil, closeError(err)
			}
			flushCandidates := proxyICECandidates(rtc, conn)
			err = rtc.SetRemoteDescription(*msg.Offer)
			if err != nil {
				return nil, closeError(fmt.Errorf("apply offer: %w", err))
			}
			answer, err := rtc.CreateAnswer(nil)
			if err != nil {
				return nil, closeError(fmt.Errorf("create answer: %w", err))
			}
			err = rtc.SetLocalDescription(answer)
			if err != nil {
				return nil, closeError(fmt.Errorf("set local answer: %w", err))
			}
			flushCandidates()

			data, err := json.Marshal(&protoMessage{
				Answer: rtc.LocalDescription(),
			})
			if err != nil {
				return nil, closeError(fmt.Errorf("marshal: %w", err))
			}
			_, err = conn.Write(data)
			if err != nil {
				return nil, closeError(fmt.Errorf("write: %w", err))
			}
		}
	}

	return nil, nil
}

func (l *listener) Close() error {
	return nil
}

func (l *listener) Addr() net.Addr {
	return nil
}
