package agent

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"cdr.dev/slog"
	"github.com/hashicorp/yamux"
	"go.coder.com/retry"
	"golang.org/x/xerrors"
	"nhooyr.io/websocket"

	"cdr.dev/coder-cli/coder-sdk"
)

const (
	listenRoute = "/api/private/envagent/listen"
)

// Server connects to a Coder deployment and listens for p2p connections.
type Server struct {
	log         slog.Logger
	listenURL   *url.URL
	coderClient coder.Client
}

// ServerArgs are the required arguments to create an agent server.
type ServerArgs struct {
	Log         slog.Logger
	CoderURL    *url.URL
	Token       string
	CoderClient coder.Client
}

// NewServer creates a new agent server.
func NewServer(args ServerArgs) (*Server, error) {
	lURL, err := formatListenURL(args.CoderURL, args.Token)
	if err != nil {
		return nil, xerrors.Errorf("formatting listen url: %w", err)
	}

	return &Server{
		log:         args.Log,
		listenURL:   lURL,
		coderClient: args.CoderClient,
	}, nil
}

// TrustCertificate will fetch coderd's certificate and write it to disc.
// It will then extend the certs to trust to include this directory.
// This only happens if coderd can answer the challenge to prove
// it has the shared secret.
func (s *Server) TrustCertificate(ctx context.Context) ([][]byte, error) {
	conf := &tls.Config{InsecureSkipVerify: true}
	hc := &http.Client{
		Timeout: time.Second * 3,
		Transport: &http.Transport{
			TLSClientConfig: conf,
		},
	}

	orig := s.coderClient.HTTPClient()
	s.coderClient.SetHTTPClient(hc)
	// Return to the original client
	defer s.coderClient.SetHTTPClient(orig)

	id := os.Getenv("CODER_WORKSPACE_ID")
	challenge, err := s.coderClient.TrustEnvironment(ctx, id)
	if err != nil {
		return nil, xerrors.Errorf("challenge failed: %w", err)
	}

	return challenge.Certificates, nil
}

// Run will listen and proxy new peer connections on a retry loop.
func (s *Server) Run(ctx context.Context) error {
	err := retry.New(time.Second).
		Context(ctx).
		Backoff(15 * time.Second).
		Conditions(
			retry.Condition(func(err error) bool {
				if err != nil {
					s.log.Error(ctx, "failed to connect", slog.Error(err))
				}
				return true
			}),
		).Run(
		func() error {
			ctx, cancelFunc := context.WithTimeout(ctx, time.Second*15)
			defer cancelFunc()
			s.log.Info(ctx, "connecting to coder", slog.F("url", s.listenURL.String()))
			conn, resp, err := websocket.Dial(ctx, s.listenURL.String(), nil)
			if err != nil && resp == nil {
				return fmt.Errorf("dial: %w", err)
			}
			if err != nil && resp != nil {
				return &coder.HTTPError{
					Response: resp,
				}
			}
			nc := websocket.NetConn(context.Background(), conn, websocket.MessageBinary)
			session, err := yamux.Server(nc, nil)
			if err != nil {
				return fmt.Errorf("open: %w", err)
			}
			s.log.Info(ctx, "connected to coder. awaiting connection requests")
			for {
				st, err := session.AcceptStream()
				if err != nil {
					return fmt.Errorf("accept stream: %w", err)
				}
				stream := &stream{
					logger: s.log.Named(fmt.Sprintf("stream %d", st.StreamID())),
					stream: st,
				}
				go stream.listen()
			}
		})

	return err
}

func formatListenURL(coderURL *url.URL, token string) (*url.URL, error) {
	if coderURL.Scheme != "http" && coderURL.Scheme != "https" {
		return nil, xerrors.Errorf("invalid URL scheme")
	}

	coderURL.Path = listenRoute
	q := coderURL.Query()
	q.Set("service_token", token)
	coderURL.RawQuery = q.Encode()

	return coderURL, nil
}
