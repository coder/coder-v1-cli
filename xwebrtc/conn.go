package xwebrtc

import (
	"context"
	"io"
	"net"
	"time"

	"github.com/pion/webrtc/v3"
	"golang.org/x/xerrors"
)

// Conn is a net.Conn based on a data channel.
type Conn struct {
	channel *webrtc.DataChannel
	rwc     io.ReadWriteCloser
}

// NewConn creates a new data channel on the peer connection and returns it as a net.Conn.
func NewConn(ctx context.Context, rtc *webrtc.PeerConnection, network string, addr string) (net.Conn, error) {
	channel, err := newProxyDataChannel(rtc, network, addr)
	if err != nil {
		return nil, xerrors.Errorf("creating data channel: %w", err)
	}
	err = waitForDataChannelOpen(ctx, channel)
	if err != nil {
		return nil, xerrors.Errorf("waiting for open data channel: %w", err)
	}

	rwc, err := channel.Detach()
	if err != nil {
		return nil, xerrors.Errorf("detaching data channel: %w", err)
	}

	return &Conn{
		channel: channel,
		rwc:     rwc,
	}, nil
}

// Read reads data from the connection.
func (c *Conn) Read(b []byte) (n int, err error) {
	return c.rwc.Read(b)
}

// Write writes data to the connection.
func (c *Conn) Write(b []byte) (n int, err error) {
	return c.rwc.Write(b)
}

// Close closes the connection.
// Any blocked Read or Write operations will be unblocked and return errors.
func (c *Conn) Close() error {
	return c.rwc.Close()
}

// LocalAddr is not implemented.
func (c *Conn) LocalAddr() net.Addr {
	return nil
}

// RemoteAddr is not implemented.
func (c *Conn) RemoteAddr() net.Addr {
	return nil
}

// SetDeadline is not implemented.
func (c *Conn) SetDeadline(t time.Time) error {
	return nil
}

// SetReadDeadline is not implemented.
func (c *Conn) SetReadDeadline(t time.Time) error {
	return nil
}

// SetWriteDeadline is not implemented.
func (c *Conn) SetWriteDeadline(t time.Time) error {
	return nil
}
