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
	"sync"
	"time"

	"github.com/pion/dtls/v2"
	"github.com/pion/ice/v2"
	"github.com/pion/logging"
	"github.com/pion/turn/v2"
	"github.com/pion/webrtc/v3"
	"golang.org/x/net/proxy"
)

var (
	// ErrMismatchedProtocol occurs when a TURN is requested to a STUN server,
	// or a TURN server is requested instead of TURNS.
	ErrMismatchedProtocol = errors.New("mismatched protocols")
	// ErrInvalidCredentials occurs when invalid credentials are passed to a
	// TURN server. This error cannot occur for STUN servers, as they don't accept
	// credentials.
	ErrInvalidCredentials = errors.New("invalid credentials")

	// Constant for the control channel protocol.
	controlChannel = "control"
)

// DialICEOptions provides options for dialing an ICE server.
type DialICEOptions struct {
	Timeout time.Duration
	// Whether to ignore TLS errors.
	InsecureSkipVerify bool
}

// DialICE confirms ICE servers are dialable.
// Timeout defaults to 200ms.
func DialICE(server webrtc.ICEServer, options *DialICEOptions) error {
	if options == nil {
		options = &DialICEOptions{}
	}

	for _, rawURL := range server.URLs {
		err := dialICEURL(server, rawURL, options)
		if err != nil {
			return err
		}
	}
	return nil
}

func dialICEURL(server webrtc.ICEServer, rawURL string, options *DialICEOptions) error {
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
				InsecureSkipVerify: options.InsecureSkipVerify,
			})
			err = dialErr
			udpConn = turn.NewSTUNConn(dconn)
		case ice.ProtoTypeTCP:
			tcpConn, err = tls.Dial("tcp4", turnServerAddr, &tls.Config{
				InsecureSkipVerify: options.InsecureSkipVerify,
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
		RTO:            options.Timeout,
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
	return nil
}

// Generalizes creating a new peer connection with consistent options.
func newPeerConnection(servers []webrtc.ICEServer, dialer proxy.Dialer) (*webrtc.PeerConnection, error) {
	se := webrtc.SettingEngine{}
	se.SetNetworkTypes([]webrtc.NetworkType{webrtc.NetworkTypeUDP4})
	se.SetSrflxAcceptanceMinWait(0)
	se.DetachDataChannels()
	// If the disconnect and keep-alive timeouts are too closely related, we'll
	// experience "random" connection failures.
	se.SetICETimeouts(time.Second*5, time.Second*25, time.Second*2)
	lf := logging.NewDefaultLoggerFactory()
	lf.DefaultLogLevel = logging.LogLevelDisabled
	se.LoggerFactory = lf

	// Enables tunneling of TURN traffic through an arbitrary proxy.
	// We proxy TURN over a WebSocket to reduce deployment complexity.
	if dialer != nil {
		se.SetICEProxyDialer(dialer)
	}

	transportPolicy := webrtc.ICETransportPolicyAll

	// If one server is provided and we know it's TURN, we can set the
	// relay acceptable so the connection starts immediately.
	if len(servers) == 1 {
		server := servers[0]
		if len(server.URLs) == 1 {
			url, err := ice.ParseURL(server.URLs[0])
			if err == nil && server.Credential != nil && url.Proto == ice.ProtoTypeTCP {
				se.SetNetworkTypes([]webrtc.NetworkType{webrtc.NetworkTypeTCP4, webrtc.NetworkTypeTCP6})
				se.SetRelayAcceptanceMinWait(0)
			}
			if err == nil && (url.Scheme == ice.SchemeTypeTURN || url.Scheme == ice.SchemeTypeTURNS) {
				// Local peers will connect if they discover they live on the same host.
				// For testing purposes, it's simpler if they cannot peer on the same host.
				transportPolicy = webrtc.ICETransportPolicyRelay
			}
		}
	}
	api := webrtc.NewAPI(webrtc.WithSettingEngine(se))

	return api.NewPeerConnection(webrtc.Configuration{
		ICEServers:         servers,
		ICETransportPolicy: transportPolicy,
	})
}

// Proxies ICE candidates using the protocol to a writer.
func proxyICECandidates(conn *webrtc.PeerConnection, w io.Writer) func() {
	var (
		mut     sync.Mutex
		queue   = []*webrtc.ICECandidate{}
		flushed = false
		write   = func(i *webrtc.ICECandidate) {
			b, _ := json.Marshal(&BrokerMessage{
				Candidate: i.ToJSON().Candidate,
			})
			_, _ = w.Write(b)
		}
	)

	conn.OnICECandidate(func(i *webrtc.ICECandidate) {
		if i == nil {
			return
		}
		mut.Lock()
		defer mut.Unlock()
		if !flushed {
			queue = append(queue, i)
			return
		}

		write(i)
	})
	return func() {
		mut.Lock()
		defer mut.Unlock()
		for _, i := range queue {
			write(i)
		}
		flushed = true
	}
}

// Waits for a PeerConnection to hit the open state.
func waitForConnectionOpen(ctx context.Context, conn *webrtc.PeerConnection) error {
	if conn.ConnectionState() == webrtc.PeerConnectionStateConnected {
		return nil
	}
	var cancel context.CancelFunc
	if _, deadlineSet := ctx.Deadline(); deadlineSet {
		ctx, cancel = context.WithCancel(ctx)
	} else {
		ctx, cancel = context.WithTimeout(ctx, time.Second*15)
	}
	defer cancel()
	conn.OnConnectionStateChange(func(pcs webrtc.PeerConnectionState) {
		if pcs == webrtc.PeerConnectionStateConnected {
			cancelFunc()
		}
	})
	<-ctx.Done()
	if ctx.Err() == context.DeadlineExceeded {
		return ctx.Err()
	}
	return nil
}

// Waits for a DataChannel to hit the open state.
func waitForDataChannelOpen(ctx context.Context, channel *webrtc.DataChannel) error {
	if channel.ReadyState() == webrtc.DataChannelStateOpen {
		return nil
	}
	if channel.ReadyState() != webrtc.DataChannelStateConnecting {
		return fmt.Errorf("channel closed")
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
