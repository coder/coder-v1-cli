package xwebrtc

import "github.com/pion/webrtc/v3"

// NewPeerConnection creates a new peer connection.
func NewPeerConnection(stunServer string) (*webrtc.PeerConnection, error) {
	se := webrtc.SettingEngine{}
	se.DetachDataChannels()
	api := webrtc.NewAPI(webrtc.WithSettingEngine(se))

	return api.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302?transport=tcp"},
			},
		},
	})
}
