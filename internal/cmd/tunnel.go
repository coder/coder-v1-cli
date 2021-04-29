package cmd

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"strconv"

	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/sloghuman"
	"github.com/spf13/cobra"
	"golang.org/x/xerrors"

	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/coder-cli/internal/x/xcobra"
	"cdr.dev/coder-cli/xwebrtc"
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

			sdk, err := newClient(ctx)
			if err != nil {
				return xerrors.Errorf("getting coder client: %w", err)
			}
			baseURL := sdk.BaseURL()

			envs, err := getEnvs(ctx, sdk, coder.Me)
			if err != nil {
				return xerrors.Errorf("get workspaces: %w", err)
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

			c := &tunnneler{
				log:         log.Leveled(slog.LevelDebug),
				brokerAddr:  &baseURL,
				token:       sdk.Token(),
				workspaceID: envID,
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
	wd, err := xwebrtc.NewWorkspaceDialer(ctx, c.log, c.brokerAddr, c.token, c.workspaceID)
	if err != nil {
		return xerrors.Errorf("creating workspace dialer: %w", wd)
	}
	nc, err := wd.DialContext(ctx, xwebrtc.NetworkTCP, fmt.Sprintf("localhost:%d", c.remotePort))
	if err != nil {
		return xerrors.Errorf("dial: %w", err)
	}

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
