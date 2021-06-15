package wsnet

import (
	"fmt"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/pion/datachannel"
	"github.com/pion/webrtc/v3"
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

// TURNEndpoint returns the TURN address for a Coder baseURL.
func TURNEndpoint(baseURL *url.URL) string {
	turnScheme := "turns"
	if baseURL.Scheme == httpScheme {
		turnScheme = "turn"
	}

	return fmt.Sprintf("%s:%s:5349?transport=tcp", turnScheme, baseURL.Hostname())
}

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
