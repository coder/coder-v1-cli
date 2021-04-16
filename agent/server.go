package agent

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"cdr.dev/slog"
	"github.com/hashicorp/yamux"
	"go.coder.com/retry"
	"golang.org/x/xerrors"
	"nhooyr.io/websocket"
)

const (
	listenRoute = "/api/private/envagent/listen"
)

// Server connects to a Coder deployment and listens for p2p connections.
type Server struct {
	log       slog.Logger
	listenURL *url.URL
}

// ServerArgs are the required arguments to create an agent server.
type ServerArgs struct {
	Log      slog.Logger
	CoderURL *url.URL
	Token    string
}

// NewServer creates a new agent server.
func NewServer(args ServerArgs) (*Server, error) {
	lURL, err := formatListenURL(args.CoderURL, args.Token)
	if err != nil {
		return nil, xerrors.Errorf("formatting listen url: %w", err)
	}

	return &Server{
		log:       args.Log,
		listenURL: lURL,
	}, nil
}

// Run will listen and proxy new peer connections on a retry loop.
func (s *Server) Run(ctx context.Context) error {
	err := retry.New(time.Second).Context(ctx).Backoff(15 * time.Second).Run(func() error {
		ctx, cancelFunc := context.WithTimeout(ctx, time.Second*15)
		defer cancelFunc()
		s.log.Info(ctx, "connecting to coder", slog.F("url", s.listenURL.String()))
		conn, _, err := websocket.Dial(ctx, s.listenURL.String(), nil)
		if err != nil {
			return fmt.Errorf("dial: %w", err)
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
