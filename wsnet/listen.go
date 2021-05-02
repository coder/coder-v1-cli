package wsnet

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"

	"cdr.dev/coder-cli/coder-sdk"
	"github.com/hashicorp/yamux"
	"github.com/pion/webrtc/v3"
	"nhooyr.io/websocket"
)

// Listen connects to the broker and returns a Listener that's triggered
// when a new connection is requested from a Dialer.
//
// LocalAddr on connections indicates the target specified by the dialer.
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
		return nil, fmt.Errorf("create multiplex: %w", err)
	}
	l := &listener{
		ws:    conn,
		conns: make(chan net.Conn),
	}
	go func() {
		for {
			conn, err := session.Accept()
			if err != nil {
				l.acceptError = err
				l.Close()
				return
			}
			l.negotiate(conn)
		}
	}()
	return l, nil
}

type listener struct {
	acceptError error
	ws          *websocket.Conn

	conns chan net.Conn
}

// Negotiates the handshake protocol over the connection provided.
func (l *listener) negotiate(conn net.Conn) {
	var (
		err     error
		decoder = json.NewDecoder(conn)
		rtc     *webrtc.PeerConnection
		// Sends the error provided then closes the connection.
		// If RTC isn't connected, we'll close it.
		closeError = func(err error) {
			d, _ := json.Marshal(&protoMessage{
				Error: err.Error(),
			})
			_, _ = conn.Write(d)
			_ = conn.Close()
			if rtc != nil {
				if rtc.ConnectionState() != webrtc.PeerConnectionStateConnected {
					rtc.Close()
					rtc = nil
				}
			}
		}
	)

	for {
		var msg protoMessage
		err = decoder.Decode(&msg)
		if err != nil {
			closeError(err)
			return
		}

		if msg.Candidate != "" {
			if rtc == nil {
				closeError(fmt.Errorf("offer must be sent before candidates"))
				return
			}

			err = rtc.AddICECandidate(webrtc.ICECandidateInit{
				Candidate: msg.Candidate,
			})
			if err != nil {
				closeError(fmt.Errorf("accept ice candidate: %w", err))
				return
			}
		}

		if msg.Offer != nil {
			if msg.Servers == nil {
				closeError(fmt.Errorf("ICEServers must be provided"))
				return
			}
			rtc, err = newPeerConnection(msg.Servers)
			if err != nil {
				closeError(err)
				return
			}
			rtc.OnDataChannel(l.handle)
			flushCandidates := proxyICECandidates(rtc, conn)
			err = rtc.SetRemoteDescription(*msg.Offer)
			if err != nil {
				closeError(fmt.Errorf("apply offer: %w", err))
				return
			}
			answer, err := rtc.CreateAnswer(nil)
			if err != nil {
				closeError(fmt.Errorf("create answer: %w", err))
				return
			}
			err = rtc.SetLocalDescription(answer)
			if err != nil {
				closeError(fmt.Errorf("set local answer: %w", err))
				return
			}
			flushCandidates()

			data, err := json.Marshal(&protoMessage{
				Answer: rtc.LocalDescription(),
			})
			if err != nil {
				closeError(fmt.Errorf("marshal: %w", err))
				return
			}
			_, err = conn.Write(data)
			if err != nil {
				closeError(fmt.Errorf("write: %w", err))
				return
			}
		}
	}
}

func (l *listener) handle(dc *webrtc.DataChannel) {
	if dc.Protocol() == controlChannel {
		// The control channel handles pings.
		dc.OnOpen(func() {
			rw, err := dc.Detach()
			if err != nil {
				return
			}
			// We'll read and write back a single byte for ping/pongin'.
			d := make([]byte, 1)
			for {
				_, err = rw.Read(d)
				if err != nil {
					continue
				}
				_, _ = rw.Write(d)
			}
		})
		return
	}

	dc.OnOpen(func() {
		rw, err := dc.Detach()
		if err != nil {
			return
		}
		parts := strings.SplitN(dc.Protocol(), ":", 2)
		network := parts[0]
		addr := parts[1]

		l.conns <- &conn{
			addr: &net.UnixAddr{
				Name: addr,
				Net:  network,
			},
			rw: rw,
		}
	})
}

// Accept accepts a new connection.
func (l *listener) Accept() (net.Conn, error) {
	return <-l.conns, l.acceptError
}

// Close closes the broker socket.
func (l *listener) Close() error {
	close(l.conns)
	return l.ws.Close(websocket.StatusNormalClosure, "")
}

// Since this listener is bound to the WebSocket, we could
// return that resolved Addr, but until we need it we won't.
func (l *listener) Addr() net.Addr {
	return nil
}
