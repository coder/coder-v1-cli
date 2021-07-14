package wsnet

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"cdr.dev/coder-cli/coder-sdk"
	"github.com/pion/datachannel"
	"github.com/pion/webrtc/v3"
	"golang.org/x/net/proxy"
	"nhooyr.io/websocket"
)

const (
	httpScheme = "http"

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

// TURNWebSocketICECandidate returns a valid relay ICEServer that can be used to
// trigger a TURNWebSocketDialer.
func TURNWebSocketICECandidate() webrtc.ICEServer {
	return webrtc.ICEServer{
		URLs:           []string{"turn:127.0.0.1:3478?transport=tcp"},
		Username:       "nop",
		Credential:     "nop",
		CredentialType: webrtc.ICECredentialTypePassword,
	}
}

// TURNWebSocketDialer proxies all TURN traffic through a WebSocket for the workspace.
func TURNWebSocketDialer(baseURL *url.URL, token string) proxy.Dialer {
	return &turnProxyDialer{
		baseURL: baseURL,
		token:   token,
	}
}

type turnProxyDialer struct {
	baseURL *url.URL
	token   string
}

func (t *turnProxyDialer) Dial(network, addr string) (c net.Conn, err error) {
	headers := http.Header{}
	headers.Set("Session-Token", t.token)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	// Copy the baseURL so we can adjust path.
	url := *t.baseURL
	url.Scheme = "wss"
	if url.Scheme == httpScheme {
		url.Scheme = "ws"
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

	return websocket.NetConn(ctx, conn, websocket.MessageBinary), nil
}

type conn struct {
	addr *net.UnixAddr
	dc   *webrtc.DataChannel
	rw   datachannel.ReadWriteCloser

	sendMore    chan struct{}
	closedMutex sync.RWMutex
	closed      bool

	writeMutex sync.Mutex
}

func (c *conn) init() {
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

func (c *conn) Read(b []byte) (n int, err error) {
	return c.rw.Read(b)
}

func (c *conn) Write(b []byte) (n int, err error) {
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

func (c *conn) Close() error {
	c.closedMutex.Lock()
	defer c.closedMutex.Unlock()
	if !c.closed {
		c.closed = true
		close(c.sendMore)
	}
	return c.dc.Close()
}

func (c *conn) LocalAddr() net.Addr {
	return c.addr
}

func (c *conn) RemoteAddr() net.Addr {
	return c.addr
}

func (c *conn) SetDeadline(t time.Time) error {
	return nil
}

func (c *conn) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *conn) SetWriteDeadline(t time.Time) error {
	return nil
}
