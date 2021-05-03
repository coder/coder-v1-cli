package wsnet

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/pion/dtls/v2"
	"github.com/pion/ice/v2"
	"github.com/pion/turn/v2"
	"github.com/pion/webrtc/v3"
)

var (
	// ErrMismatchedProtocol occurs when a TURN is requested to a STUN server,
	// or a TURN server is requested instead of TURNS.
	ErrMismatchedProtocol = errors.New("mismatched protocols")
	// ErrInvalidCredentials occurs when invalid credentials are passed to a
	// TURN server. This error cannot occur for STUN servers, as they don't accept
	// credentials.
	ErrInvalidCredentials = errors.New("invalid credentials")

	// DefaultPublicSTUN references a default STUN server we use for
	// users happy with external dependencies.
	DefaultPublicSTUN = webrtc.ICEServer{
		URLs: []string{"stun:stun.l.google.com:19302"},
	}

	// DefaultICETimeout refers to the default retry for dialing an
	// ICE server.
	DefaultICETimeout = time.Millisecond * 200

	// Constant for the control channel protocol.
	controlChannel = "control"
)

// DialICE confirms ICE servers are dialable.
// Timeout defaults to `DefaultICETimeout`.
func DialICE(server webrtc.ICEServer, timeout time.Duration) error {
	if timeout == 0 {
		timeout = DefaultICETimeout
	}

	for _, rawURL := range server.URLs {
		url, err := ice.ParseURL(rawURL)
		if err != nil {
			return err
		}
		var (
			tcpConn        net.Conn
			udpConn        net.PacketConn
			turnServerAddr = fmt.Sprintf("%s:%d", url.Host, url.Port)
		)
		switch {
		case url.Scheme == ice.SchemeTypeTURN || url.Scheme == ice.SchemeTypeSTUN:
			switch url.Proto {
			case ice.ProtoTypeUDP:
				udpConn, err = net.ListenPacket("udp4", "0.0.0.0:0")
			case ice.ProtoTypeTCP:
				tcpConn, err = net.Dial("tcp4", turnServerAddr)
			}
		case url.Scheme == ice.SchemeTypeTURNS || url.Scheme == ice.SchemeTypeSTUNS:
			switch url.Proto {
			case ice.ProtoTypeUDP:
				udpAddr, resErr := net.ResolveUDPAddr("udp4", turnServerAddr)
				if resErr != nil {
					return resErr
				}
				dconn, dialErr := dtls.Dial("udp4", udpAddr, &dtls.Config{
					InsecureSkipVerify: true,
				})
				err = dialErr
				udpConn = turn.NewSTUNConn(dconn)
			case ice.ProtoTypeTCP:
				tcpConn, err = tls.Dial("tcp4", turnServerAddr, &tls.Config{
					InsecureSkipVerify: true,
				})
			}
		}

		if err != nil {
			return err
		}
		if tcpConn != nil {
			udpConn = turn.NewSTUNConn(tcpConn)
		}
		defer udpConn.Close()

		var pass string
		if server.Credential != nil && server.CredentialType == webrtc.ICECredentialTypePassword {
			pass = server.Credential.(string)
		}

		client, err := turn.NewClient(&turn.ClientConfig{
			STUNServerAddr: turnServerAddr,
			TURNServerAddr: turnServerAddr,
			Username:       server.Username,
			Password:       pass,
			Realm:          "",
			Conn:           udpConn,
			RTO:            timeout,
		})
		if err != nil {
			return err
		}
		defer client.Close()
		err = client.Listen()
		if err != nil {
			return err
		}
		// STUN servers are not authenticated with credentials.
		// As long as the transport is valid, this should always work.
		_, err = client.SendBindingRequest()
		if err != nil {
			// Transport failed to connect.
			// https://github.com/pion/turn/blob/8231b69046f562420299916e9fb69cbff4754231/errors.go#L20
			if strings.Contains(err.Error(), "retransmissions failed") {
				return ErrMismatchedProtocol
			}
			return fmt.Errorf("binding: %w", err)
		}
		if url.Scheme == ice.SchemeTypeTURN || url.Scheme == ice.SchemeTypeTURNS {
			// We TURN to validate server credentials are correct.
			pc, err := client.Allocate()
			if err != nil {
				if strings.Contains(err.Error(), "error 400") {
					return ErrInvalidCredentials
				}
				// Since TURN and STUN follow the same protocol, they can
				// both handshake, but once a tunnel is allocated it will
				// fail to transmit.
				if strings.Contains(err.Error(), "retransmissions failed") {
					return ErrMismatchedProtocol
				}
				return err
			}
			defer pc.Close()
		}
	}
	return nil
}

// Generalizes creating a new peer connection with consistent options.
func newPeerConnection(servers []webrtc.ICEServer) (*webrtc.PeerConnection, error) {
	se := webrtc.SettingEngine{}
	se.DetachDataChannels()
	se.SetICETimeouts(time.Second*5, time.Second*5, time.Second*2)
	api := webrtc.NewAPI(webrtc.WithSettingEngine(se))

	return api.NewPeerConnection(webrtc.Configuration{
		ICEServers: servers,
	})
}

// Proxies ICE candidates using the protocol to a writer.
func proxyICECandidates(conn *webrtc.PeerConnection, w io.Writer) func() {
	queue := make([]*webrtc.ICECandidate, 0)
	flushed := false
	write := func(i *webrtc.ICECandidate) {
		b, _ := json.Marshal(&protoMessage{
			Candidate: i.ToJSON().Candidate,
		})
		_, _ = w.Write(b)
	}

	conn.OnICECandidate(func(i *webrtc.ICECandidate) {
		if i == nil {
			return
		}
		if !flushed {
			queue = append(queue, i)
			return
		}

		write(i)
	})
	return func() {
		for _, i := range queue {
			write(i)
		}
		flushed = true
	}
}

// Waits for a DataChannel to hit the open state.
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

func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}
