package xwebrtc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/url"

	"golang.org/x/net/proxy"

	"cdr.dev/slog"
	"github.com/pion/webrtc/v3"
	"golang.org/x/xerrors"
	"nhooyr.io/websocket"

	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/coder-cli/pkg/proto"
)

const (
	// NetworkTCP is the protocol for tcp tunnels.
	NetworkTCP = "tcp"
)

// WorkspaceDialer dials workspace agents and represents peer connections as a http.Client.
type WorkspaceDialer struct {
	log         slog.Logger
	brokerAddr  *url.URL
	token       string
	workspaceID string
	peerConn    *webrtc.PeerConnection
}

// NewWorkspaceDialer creates a new workspace client to dial to agents.
func NewWorkspaceDialer(ctx context.Context, log slog.Logger, brokerAddr *url.URL, token string, workspaceID string) (proxy.ContextDialer, error) {
	client := &WorkspaceDialer{
		log:         log,
		brokerAddr:  brokerAddr,
		token:       token,
		workspaceID: workspaceID,
	}

	var err error
	client.peerConn, err = client.peerConnection(ctx, workspaceID)
	if err != nil {
		return nil, xerrors.Errorf("getting peer connection: %w", err)
	}

	return client, nil
}

// DialContext will create a new peer connection with the workspace agent, make a new data channel, and return it as
// a net.Conn.
func (wc *WorkspaceDialer) DialContext(ctx context.Context, network string, workspaceAddr string) (net.Conn, error) {
	wc.log.Debug(ctx, "making net conn", slog.F("addr", workspaceAddr))
	nc, err := NewConn(ctx, wc.peerConn, network, workspaceAddr)
	if err != nil {
		return nil, xerrors.Errorf("creating net conn: %w", err)
	}

	return nc, nil
}

// peerConnection connects to a workspace agent and gives a instantiated connection with the agent.
func (wc *WorkspaceDialer) peerConnection(ctx context.Context, workspaceID string) (*webrtc.PeerConnection, error) {
	// Only enabled under a private feature flag for now,
	// so insecure connections are entirely fine to allow.
	var servers = []webrtc.ICEServer{{
		URLs:           []string{turnAddr(wc.brokerAddr)},
		Username:       "insecure",
		Credential:     "pass",
		CredentialType: webrtc.ICECredentialTypePassword,
	}}

	wc.log.Debug(ctx, "dialing broker", slog.F("url", connnectAddr(wc.brokerAddr, workspaceID, wc.token)), slog.F("servers", servers))
	conn, resp, err := websocket.Dial(ctx, connnectAddr(wc.brokerAddr, workspaceID, wc.token), nil)
	if err != nil && resp == nil {
		return nil, xerrors.Errorf("dial: %w", err)
	}
	if err != nil && resp != nil {
		defer func() {
			_ = resp.Body.Close()
		}()
		return nil, &coder.HTTPError{
			Response: resp,
		}
	}
	nconn := websocket.NetConn(ctx, conn, websocket.MessageBinary)
	defer func() {
		_ = nconn.Close()
		_ = conn.Close(websocket.StatusNormalClosure, "webrtc handshake complete")
	}()

	rtc, err := NewPeerConnection(servers)
	if err != nil {
		return nil, xerrors.Errorf("create connection: %w", err)
	}

	rtc.OnNegotiationNeeded(func() {
		wc.log.Debug(ctx, "negotiation needed...")
	})

	rtc.OnConnectionStateChange(func(pcs webrtc.PeerConnectionState) {
		wc.log.Info(ctx, "connection state changed", slog.F("state", pcs))
	})

	flushCandidates := proto.ProxyICECandidates(rtc, nconn)

	// we make a channel so the handshake actually fires
	// but we do nothing with it
	control, err := newControlDataChannel(rtc)
	if err != nil {
		return nil, xerrors.Errorf("create connect data channel: %w", err)
	}
	go func() {
		err = waitForDataChannelOpen(ctx, control)
		_ = control.Close()
		if err != nil {

			_ = conn.Close(websocket.StatusAbnormalClosure, "data channel timed out")
			return
		}
		_ = conn.Close(websocket.StatusNormalClosure, "rtc connected")
	}()

	localDesc, err := rtc.CreateOffer(&webrtc.OfferOptions{})
	if err != nil {
		return nil, xerrors.Errorf("create offer: %w", err)
	}

	err = rtc.SetLocalDescription(localDesc)
	if err != nil {
		return nil, xerrors.Errorf("set local desc: %w", err)
	}

	b, _ := json.Marshal(&proto.Message{
		Offer:   &localDesc,
		Servers: servers,
	})

	_, err = nconn.Write(b)
	if err != nil {
		return nil, xerrors.Errorf("write offer: %w", err)
	}
	flushCandidates()

	decoder := json.NewDecoder(nconn)
	for {
		var msg proto.Message
		err = decoder.Decode(&msg)
		if xerrors.Is(err, io.EOF) {
			break
		}
		if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
			break
		}
		if err != nil {
			return nil, xerrors.Errorf("read msg: %w", err)
		}
		if msg.Candidate != "" {
			wc.log.Debug(ctx, "accepted ice candidate", slog.F("candidate", msg.Candidate))
			err = proto.AcceptICECandidate(rtc, &msg)
			if err != nil {
				return nil, xerrors.Errorf("accept ice: %w", err)
			}
			continue
		}
		if msg.Answer != nil {
			wc.log.Debug(ctx, "got answer", slog.F("answer", msg.Answer))
			err = rtc.SetRemoteDescription(*msg.Answer)
			if err != nil {
				return nil, xerrors.Errorf("set remote: %w", err)
			}
			continue
		}
		if msg.Error != "" {
			return nil, xerrors.Errorf("got error: %s", msg.Error)
		}
		wc.log.Error(ctx, "unknown message", slog.F("msg", msg))
	}

	return rtc, nil
}

func turnAddr(u *url.URL) string {
	turnScheme := "turns"
	if u.Scheme == "http" {
		turnScheme = "turn"
	}
	return fmt.Sprintf("%s:%s:5349?transport=tcp", turnScheme, u.Host)
}

func connnectAddr(baseURL *url.URL, id string, token string) string {
	return fmt.Sprintf("%s%s%s%s%s", baseURL.String(), "/api/private/envagent/", id, "/connect?session_token=", token)
}
