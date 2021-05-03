package wsnet

import (
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/pion/datachannel"
)

// TURNEndpoint returns the TURN address for a Coder baseURL.
func TURNEndpoint(baseURL *url.URL) string {
	turnScheme := "turns"
	if baseURL.Scheme == "http" {
		turnScheme = "turn"
	}
	return fmt.Sprintf("%s:%s:5349?transport=tcp", turnScheme, baseURL.Host)
}

// ListenEndpoint returns the Coder endpoint to listen for workspace connections.
func ListenEndpoint(baseURL *url.URL, token string) string {
	wsScheme := "wss"
	if baseURL.Scheme == "http" {
		wsScheme = "ws"
	}
	return fmt.Sprintf("%s:%s%s?service_token=%s", wsScheme, baseURL.Host, "/api/private/envagent/listen", token)
}

// ConnectEndpoint returns the Coder endpoint to dial a connection for a workspace.
func ConnectEndpoint(baseURL *url.URL, workspace, token string) string {
	wsScheme := "wss"
	if baseURL.Scheme == "http" {
		wsScheme = "ws"
	}
	return fmt.Sprintf("%s:%s%s%s%s%s", wsScheme, baseURL.Host, "/api/private/envagent/", workspace, "/connect?session_token=", token)
}

type conn struct {
	addr *net.UnixAddr
	rw   datachannel.ReadWriteCloser
}

func (c *conn) Read(b []byte) (n int, err error) {
	return c.rw.Read(b)
}

func (c *conn) Write(b []byte) (n int, err error) {
	return c.rw.Write(b)
}

func (c *conn) Close() error {
	return c.rw.Close()
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
