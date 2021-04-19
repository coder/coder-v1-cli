package xwebrtc

import "github.com/pion/webrtc/v3"

// NewPeerConnection creates a new peer connection.
// It uses the Google stun server by default.
func NewPeerConnection() (*webrtc.PeerConnection, error) {
	se := webrtc.SettingEngine{}
	se.DetachDataChannels()
	api := webrtc.NewAPI(webrtc.WithSettingEngine(se))

	return api.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:35.239.172.66:3478?transport=tcp"},
			},
		},
	})
}
