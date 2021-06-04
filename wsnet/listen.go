package wsnet

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/bits"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/yamux"
	"github.com/pion/webrtc/v3"
	"nhooyr.io/websocket"

	"cdr.dev/coder-cli/coder-sdk"
)

var connectionRetryInterval = time.Second

var localNet = &net.IPNet{
	IP:   net.IPv4(127, 0, 0, 0),
	Mask: net.CIDRMask(8, 32),
}

// Listen connects to the broker proxies connections to the local net.
// Close will end all RTC connections.
func Listen(ctx context.Context, broker string) (io.Closer, error) {
	l := &listener{
		broker:      broker,
		connClosers: make([]io.Closer, 0),
	}
	// We do a one-off dial outside of the loop to ensure the initial
	// connection is successful. If not, there's likely an error the
	// user needs to act on.
	ch, err := l.dial(ctx)
	if err != nil {
		return nil, err
	}
	go func() {
		for {
			err := <-ch
			if errors.Is(err, io.EOF) || errors.Is(err, yamux.ErrKeepAliveTimeout) {
				// If we hit an EOF, then the connection to the broker
				// was interrupted. We'll take a short break then dial
				// again.
				ticker := time.NewTicker(connectionRetryInterval)
				for {
					select {
					case <-ticker.C:
						ch, err = l.dial(ctx)
					case <-ctx.Done():
						err = ctx.Err()
					}
					if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
						break
					}
				}
				ticker.Stop()
			}
			if err != nil {
				l.acceptError = err
				_ = l.Close()
				break
			}
		}
	}()
	return l, nil
}

type listener struct {
	broker string

	acceptError    error
	ws             *websocket.Conn
	connClosers    []io.Closer
	connClosersMut sync.Mutex
}

func (l *listener) dial(ctx context.Context) (<-chan error, error) {
	if l.ws != nil {
		_ = l.ws.Close(websocket.StatusNormalClosure, "new connection inbound")
	}
	conn, resp, err := websocket.Dial(ctx, l.broker, nil)
	if err != nil {
		if resp != nil {
			return nil, coder.NewHTTPError(resp)
		}
		return nil, err
	}
	l.ws = conn
	nconn := websocket.NetConn(ctx, conn, websocket.MessageBinary)
	config := yamux.DefaultConfig()
	config.LogOutput = io.Discard
	session, err := yamux.Server(nconn, config)
	if err != nil {
		return nil, fmt.Errorf("create multiplex: %w", err)
	}
	errCh := make(chan error)
	go func() {
		defer close(errCh)
		for {
			conn, err := session.Accept()
			if err != nil {
				errCh <- err
				break
			}
			go l.negotiate(conn)
		}
	}()
	return errCh, nil
}

// Negotiates the handshake protocol over the connection provided.
// This functions control-flow is important to readability,
// so the cognitive overload linter has been disabled.
// nolint:gocognit,nestif
func (l *listener) negotiate(conn net.Conn) {
	var (
		err     error
		decoder = json.NewDecoder(conn)
		rtc     *webrtc.PeerConnection
		// If candidates are sent before an offer, we place them here.
		// We currently have no assurances to ensure this can't happen,
		// so it's better to buffer and process than fail.
		pendingCandidates = []webrtc.ICECandidateInit{}
		// Sends the error provided then closes the connection.
		// If RTC isn't connected, we'll close it.
		closeError = func(err error) {
			d, _ := json.Marshal(&ProtoMessage{
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
		var msg ProtoMessage
		err = decoder.Decode(&msg)
		if err != nil {
			closeError(err)
			return
		}

		if msg.Candidate != "" {
			c := webrtc.ICECandidateInit{
				Candidate: msg.Candidate,
			}

			if rtc == nil {
				pendingCandidates = append(pendingCandidates, c)
				continue
			}

			err = rtc.AddICECandidate(c)
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
			for _, server := range msg.Servers {
				err = DialICE(server, nil)
				if err != nil {
					closeError(fmt.Errorf("dial server %+v: %w", server.URLs, err))
					return
				}
			}
			rtc, err = newPeerConnection(msg.Servers)
			if err != nil {
				closeError(err)
				return
			}
			rtc.OnConnectionStateChange(func(pcs webrtc.PeerConnectionState) {
				if pcs == webrtc.PeerConnectionStateConnecting {
					return
				}
				_ = conn.Close()
			})
			flushCandidates := proxyICECandidates(rtc, conn)
			l.connClosersMut.Lock()
			l.connClosers = append(l.connClosers, rtc)
			l.connClosersMut.Unlock()
			rtc.OnDataChannel(l.handle(msg))
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

			data, err := json.Marshal(&ProtoMessage{
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

			for _, candidate := range pendingCandidates {
				err = rtc.AddICECandidate(candidate)
				if err != nil {
					closeError(fmt.Errorf("add pending candidate: %w", err))
					return
				}
			}
			pendingCandidates = nil
		}
	}
}

func (l *listener) handle(msg ProtoMessage) func(dc *webrtc.DataChannel) {
	return func(dc *webrtc.DataChannel) {
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
					if errors.Is(err, io.EOF) {
						return
					}
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

			var init dialChannelMessage
			sendInitMessage := func() {
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
			}

			network, addr, err := getAddress(msg, dc.Protocol())
			if err != nil {
				init.Err = err.Error()
				sendInitMessage()
				return
			}

			conn, err := net.Dial(network, addr)
			if err != nil {
				init.Err = err.Error()
				if op, ok := err.(*net.OpError); ok {
					init.Net = op.Net
					init.Op = op.Op
				}
			}
			sendInitMessage()
			if init.Err != "" {
				return
			}
			defer conn.Close()
			defer dc.Close()

			go func() {
				_, _ = io.Copy(rw, conn)
			}()
			_, _ = io.Copy(conn, rw)
		})
	}
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

// normalizeHost converts all representations of "localhost" to "localhost".
func normalizeHost(addr string) string {
	ip := net.ParseIP(addr)
	if ip == nil {
		return addr
	}

	if localNet.Contains(ip) {
		return "localhost"
	}
	return addr
}

// getAddress parses the data channel's protocol into an address suitable for
// net.Dial. It also verifies that the ProtoMessage permits connecting to said
// address.
func getAddress(msg ProtoMessage, protocol string) (netwk, addr string, err error) {
	parts := strings.SplitN(protocol, ":", 3)
	if len(parts) != 3 {
		return "", "", fmt.Errorf("invalid dial address: %v", protocol)
	}

	var (
		network  = parts[0]
		host     = normalizeHost(parts[1])
		port     = parts[2]
		fullAddr = net.JoinHostPort(host, port)
	)
	if len(msg.Policies) == 0 {
		return network, fullAddr, nil
	}

	portParsed, err := strconv.Atoi(port)
	if err != nil || portParsed < 0 || bits.Len(uint(portParsed)) > 16 {
		return "", "", fmt.Errorf("invalid dial address %q port: %v", protocol, port)
	}
	portParsedU16 := uint16(portParsed)

	for _, p := range msg.Policies {
		if p.Network != "" && p.Network != network {
			continue
		}
		if p.Host != "" && normalizeHost(p.Host) != host {
			continue
		}
		if p.Port != 0 && p.Port != portParsedU16 {
			continue
		}

		return network, fullAddr, nil
	}

	return "", "", fmt.Errorf("connections are not permitted to %q", err)
}
