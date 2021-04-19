package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"time"

	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/sloghuman"
	"github.com/pion/webrtc/v3"
	"github.com/spf13/cobra"
	"golang.org/x/xerrors"
	"nhooyr.io/websocket"

	"cdr.dev/coder-cli/internal/x/xcobra"
	"cdr.dev/coder-cli/internal/x/xwebrtc"
	"cdr.dev/coder-cli/pkg/proto"
)

func tunnelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tunnel [workspace_name] [workspace_port] [localhost_port]",
		Args:  xcobra.ExactArgs(3),
		Short: "proxies a port on the workspace to localhost",
		Long:  "proxies a port on the workspace to localhost",
		Example: `# run a tcp tunnel from the workspace on port 3000 to localhost:3000

coder tunnel my-dev 3000 3000
`,
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			log := slog.Make(sloghuman.Sink(os.Stderr))
			log = log.Leveled(slog.LevelDebug)

			remotePort, err := strconv.ParseUint(args[1], 10, 16)
			if err != nil {
				log.Fatal(ctx, "parse remote port", slog.Error(err))
			}

			var localPort uint64
			if args[2] != "stdio" {
				localPort, err = strconv.ParseUint(args[2], 10, 16)
				if err != nil {
					log.Fatal(ctx, "parse local port", slog.Error(err))
				}
			}

			sdk, err := newClient(ctx)
			if err != nil {
				return err
			}
			baseURL := sdk.BaseURL()

			envs, err := sdk.Environments(ctx)
			if err != nil {
				return err
			}

			var envID string
			for _, env := range envs {
				if env.Name == args[0] {
					envID = env.ID
					break
				}
			}
			if envID == "" {
				return xerrors.Errorf("No workspace found by name '%s'", args[0])
			}

			c := &client{
				id:         envID,
				stdio:      args[2] == "stdio",
				localPort:  uint16(localPort),
				remotePort: uint16(remotePort),
				ctx:        context.Background(),
				logger:     log,
				brokerAddr: baseURL.String(),
				token:      sdk.Token(),
			}

			err = c.start()
			if err != nil {
				log.Fatal(ctx, err.Error())
			}

			return nil
		},
	}

	return cmd
}

type client struct {
	ctx        context.Context
	brokerAddr string
	token      string
	logger     slog.Logger
	id         string
	remotePort uint16
	localPort  uint16
	stdio      bool
}

func (c *client) start() error {
	url := fmt.Sprintf("%s%s%s%s%s", c.brokerAddr, "/api/private/envagent/", c.id, "/connect?session_token=", c.token)
	c.logger.Info(c.ctx, "connecting to broker", slog.F("url", url))

	conn, _, err := websocket.Dial(c.ctx, url, nil)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	nconn := websocket.NetConn(context.Background(), conn, websocket.MessageBinary)

	rtc, err := xwebrtc.NewPeerConnection()
	if err != nil {
		return fmt.Errorf("create connection: %w", err)
	}

	rtc.OnNegotiationNeeded(func() {
		c.logger.Debug(context.Background(), "negotiation needed...")
	})

	rtc.OnConnectionStateChange(func(pcs webrtc.PeerConnectionState) {
		c.logger.Info(context.Background(), "connection state changed", slog.F("state", pcs))
	})

	channel, err := xwebrtc.NewProxyDataChannel(rtc, "forwarder", "tcp", c.remotePort)
	if err != nil {
		return fmt.Errorf("create data channel: %w", err)
	}
	flushCandidates := proto.ProxyICECandidates(rtc, nconn)

	localDesc, err := rtc.CreateOffer(&webrtc.OfferOptions{})
	if err != nil {
		return fmt.Errorf("create offer: %w", err)
	}

	err = rtc.SetLocalDescription(localDesc)
	if err != nil {
		return fmt.Errorf("set local desc: %w", err)
	}
	flushCandidates()

	c.logger.Debug(context.Background(), "writing offer")
	b, _ := json.Marshal(&proto.Message{
		Offer: &localDesc,
	})
	_, err = nconn.Write(b)
	if err != nil {
		return fmt.Errorf("write offer: %w", err)
	}

	go func() {
		err = xwebrtc.WaitForDataChannelOpen(context.Background(), channel)
		if err != nil {
			c.logger.Fatal(context.Background(), "waiting for data channel open", slog.Error(err))
		}
		_ = conn.Close(websocket.StatusNormalClosure, "rtc connected")
	}()

	decoder := json.NewDecoder(nconn)
	for {
		var msg proto.Message
		err = decoder.Decode(&msg)
		if err == io.EOF {
			break
		}
		if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
			break
		}
		if err != nil {
			return fmt.Errorf("read msg: %w", err)
		}
		if msg.Candidate != "" {
			c.logger.Debug(context.Background(), "accepted ice candidate", slog.F("candidate", msg.Candidate))
			err = proto.AcceptICECandidate(rtc, &msg)
			if err != nil {
				return fmt.Errorf("accept ice: %w", err)
			}
		}
		if msg.Answer != nil {
			c.logger.Debug(context.Background(), "got answer", slog.F("answer", msg.Answer))
			err = rtc.SetRemoteDescription(*msg.Answer)
			if err != nil {
				return fmt.Errorf("set remote: %w", err)
			}
		}
	}

	// Once we're open... let's test out the ping.
	pingProto := "ping"
	pingChannel, err := rtc.CreateDataChannel("pinger", &webrtc.DataChannelInit{
		Protocol: &pingProto,
	})
	if err != nil {
		return fmt.Errorf("create ping channel")
	}
	pingChannel.OnOpen(func() {
		defer func() {
			_ = pingChannel.Close()
		}()
		t1 := time.Now()
		rw, _ := pingChannel.Detach()
		defer func() {
			_ = rw.Close()
		}()
		_, _ = rw.Write([]byte("hello"))
		b := make([]byte, 64)
		_, _ = rw.Read(b)
		c.logger.Info(c.ctx, "your latency directly to the agent", slog.F("ms", time.Since(t1).Milliseconds()))
	})

	if c.stdio {
		// At this point the RTC is connected and data channel is opened...
		rw, err := channel.Detach()
		if err != nil {
			return fmt.Errorf("detach channel: %w", err)
		}
		go func() {
			_, _ = io.Copy(rw, os.Stdin)
		}()
		_, err = io.Copy(os.Stdout, rw)
		if err != nil {
			return fmt.Errorf("copy: %w", err)
		}
		return nil
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", c.localPort))
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			return fmt.Errorf("accept: %w", err)
		}
		go func() {
			defer func() {
				_ = conn.Close()
			}()
			channel, err := xwebrtc.NewProxyDataChannel(rtc, "forwarder", "tcp", c.remotePort)
			if err != nil {
				c.logger.Warn(context.Background(), "create data channel for proxying", slog.Error(err))
				return
			}
			defer func() {
				_ = channel.Close()
			}()
			err = xwebrtc.WaitForDataChannelOpen(context.Background(), channel)
			if err != nil {
				c.logger.Warn(context.Background(), "wait for data channel open", slog.Error(err))
				return
			}
			rw, err := channel.Detach()
			if err != nil {
				c.logger.Warn(context.Background(), "detach channel", slog.Error(err))
				return
			}

			go func() {
				_, _ = io.Copy(conn, rw)
			}()
			_, _ = io.Copy(rw, conn)
		}()
	}
}
