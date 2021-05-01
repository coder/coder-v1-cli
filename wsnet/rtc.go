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
	"github.com/pion/turn/v2"
	"github.com/pion/webrtc/v3"
)

// ICEServer describes a single STUN and TURN server.
type ICEServer = webrtc.ICEServer

// Generalizes creating a new peer connection with consistent options.
func newPeerConnection(servers []webrtc.ICEServer) (*webrtc.PeerConnection, error) {
	se := webrtc.SettingEngine{}
	se.DetachDataChannels()
	se.SetICETimeouts(time.Second*5, time.Second*5, time.Second*2)
	api := webrtc.NewAPI(webrtc.WithSettingEngine(se))

	var (
		wg   sync.WaitGroup
		errs = make(chan error)
	)
	for _, server := range servers {
		wg.Add(1)
		server := server
		go func() {
			defer wg.Done()
			err := connectToServer(server)
			if err != nil {
				errs <- err
			}
		}()
	}
	go func() {
		wg.Wait()
		close(errs)
	}()

	err := <-errs
	if err != nil {
		return nil, err
	}

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

func connectToServer(server webrtc.ICEServer) error {
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

		if tcpConn != nil {
			udpConn = turn.NewSTUNConn(tcpConn)
		}
		if err != nil {
			return err
		}

		var pass string
		if server.CredentialType == webrtc.ICECredentialTypePassword {
			pass = server.Credential.(string)
		}

		client, err := turn.NewClient(&turn.ClientConfig{
			STUNServerAddr: turnServerAddr,
			TURNServerAddr: turnServerAddr,
			Username:       server.Username,
			Password:       pass,
			Realm:          "coder",
			Conn:           udpConn,
		})
		if err != nil {
			return err
		}
		err = client.Listen()
		if err != nil {
			return err
		}
		addr, err := client.SendBindingRequest()
		if err != nil {
			if strings.Contains(err.Error(), "retransmissions failed") {
				return errors.New("Invalid protocol")
			}
			return fmt.Errorf("binding: %w", err)
		}
		if url.Scheme == ice.SchemeTypeTURN || url.Scheme == ice.SchemeTypeTURNS {
			pc, err := client.Allocate()
			if err != nil {
				fmt.Printf("ERR %T\n", err)
				return err
			}
			pc.Close()
		}
		fmt.Printf("Got: %+v\n", addr)
		udpConn.Close()
	}
	return nil
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
