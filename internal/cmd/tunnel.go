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
	"github.com/fatih/color"
	"github.com/pion/webrtc/v3"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"golang.org/x/xerrors"

	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/coder-cli/internal/x/xcobra"
	"cdr.dev/coder-cli/pkg/clog"
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

			workspace, err := findWorkspace(ctx, sdk, args[0], coder.Me)
			if err != nil {
				return xerrors.Errorf("get workspaces: %w", err)
			}

			if workspace.LatestStat.ContainerStatus != coder.WorkspaceOn {
				color.NoColor = false
				notAvailableError := clog.Error("workspace not available",
					fmt.Sprintf("current status: \"%s\"", workspace.LatestStat.ContainerStatus),
					clog.BlankLine,
					clog.Tipf("use \"coder workspaces rebuild %s\" to rebuild this workspace", workspace.Name),
				)
				// If we're attempting to forward our remote SSH port,
				// we want to communicate with the OpenSSH protocol so
				// SSH clients can properly display output to our users.
				if remotePort == 12213 {
					rawKey, err := sdk.SSHKey(ctx)
					if err != nil {
						return xerrors.Errorf("get ssh key: %w", err)
					}
					err = discardSSHConnection(&stdioConn{}, rawKey.PrivateKey, notAvailableError.String())
					if err != nil {
						return err
					}
					return nil
				}

				return notAvailableError
			}

			iceServers, err := sdk.ICEServers(ctx)
			if err != nil {
				return xerrors.Errorf("get ICE servers: %w", err)
			}
			log.Debug(ctx, "got ICE servers", slog.F("ice", iceServers))

			c := &tunnneler{
				log:        log,
				brokerAddr: &baseURL,
				token:      sdk.Token(),
				workspace:  workspace,
				iceServers: iceServers,
				stdio:      args[2] == "stdio",
				localPort:  uint16(localPort),
				remotePort: uint16(remotePort),
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
	log        slog.Logger
	brokerAddr *url.URL
	token      string
	workspace  *coder.Workspace
	iceServers []webrtc.ICEServer
	remotePort uint16
	localPort  uint16
	stdio      bool
}

func (c *tunnneler) start(ctx context.Context) error {
	c.log.Debug(ctx, "Connecting to workspace...")

	dialLog := c.log.Named("wsnet")
	wd, err := wsnet.DialWebsocket(
		ctx,
		wsnet.ConnectEndpoint(c.brokerAddr, c.workspace.ID, c.token),
		&wsnet.DialOptions{
			Log:                &dialLog,
			TURNProxyAuthToken: c.token,
			TURNRemoteProxyURL: c.brokerAddr,
			TURNLocalProxyURL:  c.brokerAddr,
			ICEServers:         c.iceServers,
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
				_ = sdk.UpdateLastConnectionAt(ctx, c.workspace.ID)
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

// Used to treat stdio like a connection for proxying SSH.
type stdioConn struct{}

func (s *stdioConn) Read(b []byte) (n int, err error) {
	return os.Stdin.Read(b)
}

func (s *stdioConn) Write(b []byte) (n int, err error) {
	return os.Stdout.Write(b)
}

func (s *stdioConn) Close() error {
	return nil
}

func (s *stdioConn) LocalAddr() net.Addr {
	return nil
}

func (s *stdioConn) RemoteAddr() net.Addr {
	return nil
}

func (s *stdioConn) SetDeadline(t time.Time) error {
	return nil
}

func (s *stdioConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (s *stdioConn) SetWriteDeadline(t time.Time) error {
	return nil
}

// discardSSHConnection accepts a connection then outputs the message provided
// to any channel opened, immediately closing the connection afterwards.
//
// Used to provide status to connecting clients while still aligning with the
// native SSH protocol.
func discardSSHConnection(nc net.Conn, privateKey string, msg string) error {
	config := &ssh.ServerConfig{
		NoClientAuth: true,
	}
	key, err := ssh.ParseRawPrivateKey([]byte(privateKey))
	if err != nil {
		return fmt.Errorf("parse private key: %w", err)
	}
	signer, err := ssh.NewSignerFromKey(key)
	if err != nil {
		return fmt.Errorf("signer from private key: %w", err)
	}
	config.AddHostKey(signer)
	conn, chans, reqs, err := ssh.NewServerConn(nc, config)
	if err != nil {
		return fmt.Errorf("create server conn: %w", err)
	}
	go ssh.DiscardRequests(reqs)
	ch, req, err := (<-chans).Accept()
	if err != nil {
		return fmt.Errorf("accept channel: %w", err)
	}
	go ssh.DiscardRequests(req)

	_, err = ch.Write([]byte(msg))
	if err != nil {
		return fmt.Errorf("write channel: %w", err)
	}
	err = ch.Close()
	if err != nil {
		return fmt.Errorf("close channel: %w", err)
	}
	return conn.Close()
}
