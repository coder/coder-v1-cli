package xwebrtc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"golang.org/x/xerrors"

	"github.com/pion/webrtc/v3"
)

// ParseProxyDataChannel parses a data channel to get the network and addr.
func ParseProxyDataChannel(channel *webrtc.DataChannel) (string, string, error) {
	if channel.Protocol() == "" {
		return "", "", xerrors.New("data channel is not a proxy")
	}
	segments := strings.SplitN(channel.Protocol(), ":", 2)
	if len(segments) != 2 {
		return "", "", xerrors.Errorf("protocol is malformed: %s", channel.Protocol())
	}

	return segments[0], segments[1], nil
}

// NewPeerConnection creates a new peer connection.
// It uses the Google stun server by default.
func NewPeerConnection(servers []webrtc.ICEServer) (*webrtc.PeerConnection, error) {
	se := webrtc.SettingEngine{}
	se.DetachDataChannels()
	se.SetICETimeouts(time.Second*5, time.Second*5, time.Second*2)
	api := webrtc.NewAPI(webrtc.WithSettingEngine(se))

	return api.NewPeerConnection(webrtc.Configuration{
		ICEServers: servers,
	})
}

// waitForDataChannelOpen waits for the data channel to have the open state.
// By default, it waits 15 seconds.
func waitForDataChannelOpen(ctx context.Context, channel *webrtc.DataChannel) error {
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

// newProxyDataChannel creates a new data channel for proxying.
func newProxyDataChannel(conn *webrtc.PeerConnection, protocol string, addr string) (*webrtc.DataChannel, error) {
	proto := fmt.Sprintf("%s:%s", protocol, addr)
	ordered := true
	return conn.CreateDataChannel(proto, &webrtc.DataChannelInit{
		Protocol: &proto,
		Ordered:  &ordered,
	})
}

// newControlDataChannel creates a new data channel for starting a new peer connection.
func newControlDataChannel(conn *webrtc.PeerConnection) (*webrtc.DataChannel, error) {
	proto := "control"
	ordered := true
	return conn.CreateDataChannel(proto, &webrtc.DataChannelInit{
		Protocol: &proto,
		Ordered:  &ordered,
	})
}
