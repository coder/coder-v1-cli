package wsnet

import (
	"net"
	"time"

	"github.com/pion/datachannel"
)

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
