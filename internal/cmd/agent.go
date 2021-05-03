package cmd

import (
	"context"
	"io"
	"net"
	"net/url"
	"os"
	"time"

	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/sloghuman"
	"github.com/spf13/cobra"
	"go.coder.com/retry"
	"golang.org/x/xerrors"

	"cdr.dev/coder-cli/wsnet"
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
		token    string
		coderURL string
	)
	cmd := &cobra.Command{
		Use:   "start --coder-url=[coder_url] --token=[token]",
		Short: "starts the coder agent",
		Long:  "starts the coder agent",
		Example: `# start the agent and use CODER_URL and CODER_AGENT_TOKEN env vars

coder agent start

# start the agent and connect with a specified url and agent token

coder agent start --coder-url https://my-coder.com --token xxxx-xxxx
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			log := slog.Make(sloghuman.Sink(cmd.OutOrStdout()))

			if coderURL == "" {
				var ok bool
				coderURL, ok = os.LookupEnv("CODER_URL")
				if !ok {
					client, err := newClient(ctx)
					if err != nil {
						return xerrors.New("must login, pass --coder-url flag, or set the CODER_URL env variable")
					}
					burl := client.BaseURL()
					coderURL = burl.String()
				}
			}

			u, err := url.Parse(coderURL)
			if err != nil {
				return xerrors.Errorf("parse url: %w", err)
			}

			if token == "" {
				var ok bool
				token, ok = os.LookupEnv("CODER_AGENT_TOKEN")
				if !ok {
					return xerrors.New("must pass --token or set the CODER_AGENT_TOKEN env variable")
				}
			}

			return runAgentRetry(ctx, log, u, token)
		},
	}

	cmd.Flags().StringVar(&token, "token", "", "coder agent token")
	cmd.Flags().StringVar(&coderURL, "coder-url", "", "coder access url")

	return cmd
}

func runAgentRetry(ctx context.Context, logger slog.Logger, u *url.URL, token string) error {
	return retry.New(time.Second).
		Context(ctx).
		Backoff(15 * time.Second).
		Conditions(
			retry.Condition(func(err error) bool {
				if err != nil {
					logger.Error(ctx, "failed to connect", slog.Error(err))
				}
				return true
			}),
		).Run(
		func() error {
			listener, err := wsnet.Listen(context.Background(), wsnet.ListenEndpoint(u, token))
			if err != nil {
				return xerrors.Errorf("listen: %w", err)
			}
			for {
				conn, err := listener.Accept()
				if err != nil {
					return xerrors.Errorf("accept: %w", err)
				}
				if conn.LocalAddr().Network() != "tcp" {
					logger.Warn(ctx, "client requested unsupported protocol", slog.F("protocol", conn.LocalAddr().Network()))
					conn.Close()
					continue
				}
				nconn, err := net.Dial(conn.LocalAddr().Network(), conn.LocalAddr().String())
				if err != nil {
					logger.Warn(ctx, "client dial error", slog.Error(err))
					conn.Close()
					continue
				}
				go func() {
					_, _ = io.Copy(nconn, conn)
				}()
				go func() {
					_, _ = io.Copy(conn, nconn)
				}()
			}
		})

}
