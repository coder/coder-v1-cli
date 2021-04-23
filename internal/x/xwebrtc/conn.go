package xwebrtc

import (
	"time"

	"github.com/pion/webrtc/v3"
)

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
