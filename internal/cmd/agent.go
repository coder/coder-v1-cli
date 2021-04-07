package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/sloghuman"
	"github.com/hashicorp/yamux"
	"github.com/pion/webrtc/v3"
	"github.com/spf13/cobra"
	"golang.org/x/xerrors"
	"nhooyr.io/websocket"

	"cdr.dev/coder-cli/internal/x/xcobra"
	"cdr.dev/coder-cli/internal/x/xwebrtc"
	"cdr.dev/coder-cli/pkg/proto"
)

func agentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "agent",
		Short:  "Run the workspace agent",
		Long:   "Connect to Coder and start running a p2p agent",
		Hidden: true,
	}

	cmd.AddCommand(
		startCmd(),
	)
	return cmd
}

func startCmd() *cobra.Command {
	var (
		token string
	)
	cmd := &cobra.Command{
		Use:   "start [coderURL] --token=[token]",
		Args:  xcobra.ExactArgs(1),
		Short: "starts the coder agent",
		Long:  "starts the coder agent",
		Example: `# start the agent and connect with a Coder agent token

coder agent start https://my-coder.com --token xxxx-xxxx

# start the agent and use CODER_AGENT_TOKEN env var for auth token

coder agent start https://my-coder.com
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			log := slog.Make(sloghuman.Sink(cmd.OutOrStdout()))

			// Pull the URL from the args and do some sanity check.
			rawURL := args[0]
			if rawURL == "" || !strings.HasPrefix(rawURL, "http") {
				return xerrors.Errorf("invalid URL")
			}
			u, err := url.Parse(rawURL)
			if err != nil {
				return xerrors.Errorf("parse url: %w", err)
			}
			// Remove the trailing '/' if any.
			u.Path = "/api/private/envagent/listen"

			if token == "" {
				var ok bool
				token, ok = os.LookupEnv("CODER_AGENT_TOKEN")
				if !ok {
					return xerrors.New("must pass --token or set the CODER_AGENT_TOKEN env variable")
				}
			}

			q := u.Query()
			q.Set("service_token", token)
			u.RawQuery = q.Encode()

			ctx, cancelFunc := context.WithTimeout(ctx, time.Second*15)
			defer cancelFunc()
			log.Info(ctx, "connecting to broker", slog.F("url", u.String()))
			conn, res, err := websocket.Dial(ctx, u.String(), nil)
			if err != nil {
				return fmt.Errorf("dial: %w", err)
			}
			_ = res.Body.Close()
			nc := websocket.NetConn(context.Background(), conn, websocket.MessageBinary)
			session, err := yamux.Server(nc, nil)
			if err != nil {
				return fmt.Errorf("open: %w", err)
			}
			log.Info(ctx, "connected to broker. awaiting connection requests")
			for {
				st, err := session.AcceptStream()
				if err != nil {
					return fmt.Errorf("accept stream: %w", err)
				}
				stream := &stream{
					logger: log.Named(fmt.Sprintf("stream %d", st.StreamID())),
					stream: st,
				}
				go stream.listen()
			}
		},
	}

	cmd.Flags().StringVar(&token, "token", "", "coder agent token")
	return cmd
}

type stream struct {
	stream *yamux.Stream
	logger slog.Logger

	rtc *webrtc.PeerConnection
}

// writes an error and closes.
func (s *stream) fatal(err error) {
	_ = s.write(proto.Message{
		Error: err.Error(),
	})
	s.logger.Error(context.Background(), err.Error(), slog.Error(err))
	_ = s.stream.Close()
}

func (s *stream) listen() {
	decoder := json.NewDecoder(s.stream)
	for {
		var msg proto.Message
		err := decoder.Decode(&msg)
		if err == io.EOF {
			break
		}
		if err != nil {
			s.fatal(err)
			return
		}
		s.processMessage(msg)
	}
}

func (s *stream) write(msg proto.Message) error {
	d, err := json.Marshal(&msg)
	if err != nil {
		return err
	}
	_, err = s.stream.Write(d)
	if err != nil {
		return err
	}
	return nil
}

func (s *stream) processMessage(msg proto.Message) {
	s.logger.Debug(context.Background(), "processing message", slog.F("msg", msg))

	if msg.Error != "" {
		s.fatal(xerrors.New(msg.Error))
		return
	}

	if msg.Candidate != "" {
		if s.rtc == nil {
			s.fatal(xerrors.New("rtc connection must be started before candidates are sent"))
			return
		}

		s.logger.Debug(context.Background(), "accepted ice candidate", slog.F("candidate", msg.Candidate))
		err := proto.AcceptICECandidate(s.rtc, &msg)
		if err != nil {
			s.fatal(err)
			return
		}
	}

	if msg.Offer != nil {
		rtc, err := xwebrtc.NewPeerConnection()
		if err != nil {
			s.fatal(fmt.Errorf("create connection: %w", err))
			return
		}
		flushCandidates := proto.ProxyICECandidates(rtc, s.stream)

		err = rtc.SetRemoteDescription(*msg.Offer)
		if err != nil {
			s.fatal(fmt.Errorf("set remote desc: %w", err))
			return
		}
		answer, err := rtc.CreateAnswer(nil)
		if err != nil {
			s.fatal(fmt.Errorf("create answer: %w", err))
			return
		}
		err = rtc.SetLocalDescription(answer)
		if err != nil {
			s.fatal(fmt.Errorf("set local desc: %w", err))
			return
		}
		flushCandidates()

		err = s.write(proto.Message{
			Answer: rtc.LocalDescription(),
		})
		if err != nil {
			s.fatal(fmt.Errorf("send local desc: %w", err))
			return
		}

		rtc.OnConnectionStateChange(func(pcs webrtc.PeerConnectionState) {
			s.logger.Info(context.Background(), "state changed", slog.F("new", pcs))
		})
		rtc.OnDataChannel(s.processDataChannel)
		s.rtc = rtc
	}
}

func (s *stream) processDataChannel(channel *webrtc.DataChannel) {
	if channel.Protocol() == "ping" {
		channel.OnOpen(func() {
			rw, err := channel.Detach()
			if err != nil {
				return
			}
			d := make([]byte, 64)
			_, _ = rw.Read(d)
			_, _ = rw.Write(d)
		})
		return
	}

	prto, port, err := xwebrtc.ParseProxyDataChannel(channel)
	if err != nil {
		s.fatal(fmt.Errorf("failed to parse proxy data channel: %w", err))
		return
	}
	if prto != "tcp" {
		s.fatal(fmt.Errorf("client provided unsupported protocol: %s", prto))
		return
	}

	conn, err := net.Dial(prto, fmt.Sprintf("localhost:%d", port))
	if err != nil {
		s.fatal(fmt.Errorf("failed to dial client port: %d", port))
		return
	}

	channel.OnOpen(func() {
		s.logger.Debug(context.Background(), "proxying data channel to local port", slog.F("port", port))
		rw, err := channel.Detach()
		if err != nil {
			_ = channel.Close()
			s.logger.Error(context.Background(), "detach client data channel", slog.Error(err))
			return
		}
		go func() {
			_, _ = io.Copy(rw, conn)
		}()
		go func() {
			_, _ = io.Copy(conn, rw)
		}()
	})
}
