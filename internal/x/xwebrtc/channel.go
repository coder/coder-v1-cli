package xwebrtc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/pion/webrtc/v3"
)

// WaitForDataChannelOpen waits for the data channel to have the open state.
// By default, it waits 15 seconds.
func WaitForDataChannelOpen(ctx context.Context, channel *webrtc.DataChannel) error {
	if channel.ReadyState() == webrtc.DataChannelStateOpen {
		return nil
	}
	ctx, cancelFunc := context.WithTimeout(ctx, time.Second*15)
	defer cancelFunc()
	channel.OnOpen(func() {
		cancelFunc()
	})
	<-ctx.Done()
	if ctx.Err() == context.DeadlineExceeded {
		return ctx.Err()
	}
	return nil
}

// NewProxyDataChannel creates a new data channel for proxying.
func NewProxyDataChannel(conn *webrtc.PeerConnection, name, protocol string, port uint16) (*webrtc.DataChannel, error) {
	proto := fmt.Sprintf("%s:%d", protocol, port)
	ordered := true
	return conn.CreateDataChannel(name, &webrtc.DataChannelInit{
		Protocol: &proto,
		Ordered:  &ordered,
	})
}

// ParseProxyDataChannel parses a data channel to get the protocol and port.
func ParseProxyDataChannel(channel *webrtc.DataChannel) (string, uint16, error) {
	if channel.Protocol() == "" {
		return "", 0, errors.New("data channel is not a proxy")
	}
	host, port, err := net.SplitHostPort(channel.Protocol())
	if err != nil {
		return "", 0, fmt.Errorf("split protocol: %w", err)
	}
	p, err := strconv.ParseInt(port, 10, 16)
	if err != nil {
		return "", 0, fmt.Errorf("parse port: %w", err)
	}
	return host, uint16(p), nil
}
