package cmd

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"strconv"
	"time"

	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/sloghuman"
	"github.com/pion/webrtc/v3"
	"github.com/spf13/cobra"
	"golang.org/x/xerrors"

	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/coder-cli/internal/x/xcobra"
	"cdr.dev/coder-cli/wsnet"
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
			if os.Getenv("CODER_TUNNEL_DEBUG") != "" {
				log = log.Leveled(slog.LevelDebug)
				log.Info(ctx, "debug logging enabled")
			}

			remotePort, err := strconv.ParseUint(args[1], 10, 16)
			if err != nil {
				return xerrors.Errorf("parse remote port: %w", err)
			}

			var localPort uint64
			if args[2] != "stdio" {
				localPort, err = strconv.ParseUint(args[2], 10, 16)
				if err != nil {
					return xerrors.Errorf("parse local port: %w", err)
				}
			}

			sdk, err := newClient(ctx, false)
			if err != nil {
				return xerrors.Errorf("getting coder client: %w", err)
			}
			baseURL := sdk.BaseURL()

			workspaces, err := getWorkspaces(ctx, sdk, coder.Me)
			if err != nil {
				return xerrors.Errorf("get workspaces: %w", err)
			}

			var workspaceID string
			for _, workspace := range workspaces {
				if workspace.Name == args[0] {
					workspaceID = workspace.ID
					break
				}
			}
			if workspaceID == "" {
				return xerrors.Errorf("No workspace found by name '%s'", args[0])
			}

			c := &tunnneler{
				log:         log,
				brokerAddr:  &baseURL,
				token:       sdk.Token(),
				workspaceID: workspaceID,
				stdio:       args[2] == "stdio",
				localPort:   uint16(localPort),
				remotePort:  uint16(remotePort),
			}

			err = c.start(ctx)
			if err != nil {
				return xerrors.Errorf("running tunnel: %w", err)
			}

			return nil
		},
	}

	return cmd
}

type tunnneler struct {
	log         slog.Logger
	brokerAddr  *url.URL
	token       string
	workspaceID string
	remotePort  uint16
	localPort   uint16
	stdio       bool
}

func (c *tunnneler) start(ctx context.Context) error {
	c.log.Debug(ctx, "Connecting to workspace...")

	dialLog := c.log.Named("wsnet")
	wd, err := wsnet.DialWebsocket(
		ctx,
		wsnet.ConnectEndpoint(c.brokerAddr, c.workspaceID, c.token),
		&wsnet.DialOptions{
			Log:                &dialLog,
			TURNProxyAuthToken: c.token,
			TURNRemoteProxyURL: c.brokerAddr,
			TURNLocalProxyURL:  c.brokerAddr,
			ICEServers:         []webrtc.ICEServer{wsnet.TURNProxyICECandidate()},
		},
		nil,
	)
	if err != nil {
		return xerrors.Errorf("creating workspace dialer: %w", err)
	}
	nc, err := wd.DialContext(ctx, "tcp", fmt.Sprintf("localhost:%d", c.remotePort))
	if err != nil {
		return err
	}
	c.log.Debug(ctx, "Connected to workspace!")

	sdk, err := newClient(ctx, false)
	if err != nil {
		return xerrors.Errorf("getting coder client: %w", err)
	}

	// regularly update the last connection at
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// silently ignore failures so we don't spam the console
				_ = sdk.UpdateLastConnectionAt(ctx, c.workspaceID)
			}
		}
	}()

	// proxy via stdio
	if c.stdio {
		go func() {
			_, _ = io.Copy(nc, os.Stdin)
		}()
		_, err = io.Copy(os.Stdout, nc)
		if err != nil {
			return xerrors.Errorf("copy: %w", err)
		}
		return nil
	}
	// This was used to test if the port was open, and proxy over stdio
	// if the user specified that.
	_ = nc.Close()

	// proxy via tcp listener
	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", c.localPort))
	if err != nil {
		return xerrors.Errorf("listen: %w", err)
	}

	for {
		lc, err := listener.Accept()
		if err != nil {
			return xerrors.Errorf("accept: %w", err)
		}
		nc, err := wd.DialContext(ctx, "tcp", fmt.Sprintf("localhost:%d", c.remotePort))
		if err != nil {
			return err
		}
		go func() {
			defer func() {
				_ = lc.Close()
			}()

			go func() {
				_, _ = io.Copy(lc, nc)
			}()
			_, _ = io.Copy(nc, lc)
		}()
	}
}
