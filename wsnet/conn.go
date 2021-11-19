package wsnet

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/pion/datachannel"
	"github.com/pion/webrtc/v3"
	"nhooyr.io/websocket"

	"cdr.dev/coder-cli/coder-sdk"
)

const (
	httpScheme             = "http"
	turnProxyMagicUsername = "~magicalusername~"

	bufferedAmountLowThreshold uint64 = 512 * 1024  // 512 KB
	maxBufferedAmount          uint64 = 1024 * 1024 // 1 MB
	// For some reason messages larger just don't work...
	// This shouldn't be a huge deal for real-world usage.
	// See: https://github.com/pion/datachannel/issues/59
	maxMessageLength = 32 * 1024 // 32 KB
)

// ListenEndpoint returns the Coder endpoint to listen for workspace connections.
func ListenEndpoint(baseURL *url.URL, token string) string {
	wsScheme := "wss"
	if baseURL.Scheme == httpScheme {
		wsScheme = "ws"
	}
	return fmt.Sprintf("%s://%s%s?service_token=%s", wsScheme, baseURL.Host, "/api/private/envagent/listen", token)
}

// ConnectEndpoint returns the Coder endpoint to dial a connection for a workspace.
func ConnectEndpoint(baseURL *url.URL, workspace, token string) string {
	wsScheme := "wss"
	if baseURL.Scheme == httpScheme {
		wsScheme = "ws"
	}
	return fmt.Sprintf("%s://%s%s%s%s%s", wsScheme, baseURL.Host, "/api/private/envagent/", workspace, "/connect?session_token=", token)
}

// TURNWebSocketICECandidate returns a fake TCP relay ICEServer.
// It's used to trigger the ICEProxyDialer.
func TURNProxyICECandidate() webrtc.ICEServer {
	return webrtc.ICEServer{
		URLs:           []string{"turn:127.0.0.1:3478?transport=tcp"},
		Username:       turnProxyMagicUsername,
		Credential:     turnProxyMagicUsername,
		CredentialType: webrtc.ICECredentialTypePassword,
	}
}

// Proxies all TURN ICEServer traffic through this dialer.
// References Coder APIs with a specific token.
type turnProxyDialer struct {
	baseURL *url.URL
	token   string
}

func (t *turnProxyDialer) Dial(_, _ string) (c net.Conn, err error) {
	headers := http.Header{}
	headers.Set("Session-Token", t.token)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	// Copy the baseURL so we can adjust path.
	url := *t.baseURL
	switch url.Scheme {
	case "http":
		url.Scheme = "ws"
	case "https":
		url.Scheme = "wss"
	default:
		return nil, errors.New("invalid turn url addr scheme provided")
	}
	url.Path = "/api/private/turn"
	conn, resp, err := websocket.Dial(ctx, url.String(), &websocket.DialOptions{
		HTTPHeader: headers,
	})
	if err != nil {
		if resp != nil {
			defer resp.Body.Close()
			return nil, coder.NewHTTPError(resp)
		}
		return nil, fmt.Errorf("dial: %w", err)
	}

	return &turnProxyConn{
		websocket.NetConn(context.Background(), conn, websocket.MessageBinary),
	}, nil
}

// turnProxyConn is a net.Conn wrapper that returns a TCPAddr for the
// LocalAddr function. pion/ice unsafely checks the types. See:
// https://github.com/pion/ice/blob/e78f26fb435987420546c70369ade5d713beca39/gather.go#L448
type turnProxyConn struct {
	net.Conn
}

// The LocalAddr specified here doesn't really matter,
// it just has to be of type "TCPAddr".
func (*turnProxyConn) LocalAddr() net.Addr {
	return &net.TCPAddr{
		IP:   net.IPv4(127, 0, 0, 1),
		Port: 0,
	}
}

// Properly buffers data for data channel connections.
type dataChannelConn struct {
	addr *net.UnixAddr
	dc   *webrtc.DataChannel
	rw   datachannel.ReadWriteCloser

	sendMore    chan struct{}
	closedMutex sync.RWMutex
	closed      bool

	writeMutex sync.Mutex
}

func (c *dataChannelConn) init() {
	c.closedMutex.Lock()
	defer c.closedMutex.Unlock()
	c.sendMore = make(chan struct{}, 1)
	c.dc.SetBufferedAmountLowThreshold(bufferedAmountLowThreshold)
	c.dc.OnBufferedAmountLow(func() {
		c.closedMutex.RLock()
		defer c.closedMutex.RUnlock()
		if c.closed {
			return
		}
		select {
		case c.sendMore <- struct{}{}:
		default:
		}
	})
}

func (c *dataChannelConn) Read(b []byte) (n int, err error) {
	return c.rw.Read(b)
}

func (c *dataChannelConn) Write(b []byte) (n int, err error) {
	c.writeMutex.Lock()
	defer c.writeMutex.Unlock()
	if len(b) > maxMessageLength {
		return 0, fmt.Errorf("outbound packet larger than maximum message size: %d", maxMessageLength)
	}
	if c.dc.BufferedAmount()+uint64(len(b)) >= maxBufferedAmount {
		<-c.sendMore
	}
	// TODO (@kyle): There's an obvious race-condition here.
	// This is an edge-case, as most-frequently data won't
	// be pooled so synchronously, but is definitely possible.
	//
	// See: https://github.com/pion/sctp/issues/181
	time.Sleep(time.Microsecond)

	return c.rw.Write(b)
}

func (c *dataChannelConn) Close() error {
	c.closedMutex.Lock()
	defer c.closedMutex.Unlock()
	if !c.closed {
		c.closed = true
		close(c.sendMore)
	}
	return c.dc.Close()
}

func (c *dataChannelConn) LocalAddr() net.Addr {
	return c.addr
}

func (c *dataChannelConn) RemoteAddr() net.Addr {
	return c.addr
}

func (c *dataChannelConn) SetDeadline(_ time.Time) error {
	return nil
}

func (c *dataChannelConn) SetReadDeadline(_ time.Time) error {
	return nil
}

func (c *dataChannelConn) SetWriteDeadline(_ time.Time) error {
	return nil
}
