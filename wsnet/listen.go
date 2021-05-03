package wsnet

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"

	"github.com/hashicorp/yamux"
	"github.com/pion/webrtc/v3"
	"nhooyr.io/websocket"

	"cdr.dev/coder-cli/coder-sdk"
)

// Listen connects to the broker proxies connections to the local net.
// Close will end all RTC connections.
func Listen(ctx context.Context, broker string) (io.Closer, error) {
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
		ws:          conn,
		connClosers: make([]io.Closer, 0),
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
	acceptError    error
	ws             *websocket.Conn
	connClosers    []io.Closer
	connClosersMut sync.Mutex
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
			l.connClosersMut.Lock()
			l.connClosers = append(l.connClosers, rtc)
			l.connClosersMut.Unlock()
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

		var init dialChannelMessage
		conn, err := net.Dial(network, addr)
		if err != nil {
			init.Err = err.Error()
			if op, ok := err.(*net.OpError); ok {
				init.Net = op.Net
				init.Op = op.Op
			}
		}
		initData, err := json.Marshal(&init)
		if err != nil {
			rw.Close()
			return
		}
		_, err = rw.Write(initData)
		if err != nil {
			return
		}
		if init.Err != "" {
			// If an error occurred, we're safe to close the connection.
			dc.Close()
			return
		}
		defer conn.Close()
		defer dc.Close()

		go func() {
			_, _ = io.Copy(conn, rw)
		}()
		_, _ = io.Copy(rw, conn)
	})
}

// Close closes the broker socket and all created RTC connections.
func (l *listener) Close() error {
	l.connClosersMut.Lock()
	for _, connCloser := range l.connClosers {
		// We can ignore the error here... it doesn't
		// really matter if these fail to close.
		_ = connCloser.Close()
	}
	l.connClosersMut.Unlock()
	return l.ws.Close(websocket.StatusNormalClosure, "")
}

// Since this listener is bound to the WebSocket, we could
// return that resolved Addr, but until we need it we won't.
func (l *listener) Addr() net.Addr {
	return nil
}
