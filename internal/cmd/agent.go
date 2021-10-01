package cmd

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/sloghuman"
	"github.com/spf13/cobra"
	"golang.org/x/xerrors"

	"cdr.dev/coder-cli/agent"
)

// coderdCertDir is where the certificates for coderd are written by the coder agent.
const coderdCertDir = "/var/tmp/coder/certs"

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

			c, err := newClient(ctx)
			if err != nil {
				return xerrors.Errorf("coder api client: %w", err)
			}

			server, err := agent.NewServer(agent.ServerArgs{
				Log:         log,
				CoderURL:    u,
				Token:       token,
				CoderClient: c,
			})
			if err != nil {
				return xerrors.Errorf("creating agent server: %w", err)
			}

			// Inject coderd cert incase we don't have it.
			err = writeCoderdCerts(ctx, server)
			if err != nil {
				return xerrors.Errorf("coderd cert: %w", err)
			}

			err = server.Run(ctx)
			if err != nil && !xerrors.Is(err, context.Canceled) && !xerrors.Is(err, context.DeadlineExceeded) {
				return xerrors.Errorf("running agent server: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&token, "token", "", "coder agent token")
	cmd.Flags().StringVar(&coderURL, "coder-url", "", "coder access url")

	return cmd
}

func writeCoderdCerts(ctx context.Context, srv *agent.Server) error {
	// Inject certs to custom dir and concat with : with existing dir.
	certs, err := srv.TrustCertificate(ctx)
	if err != nil {
		return xerrors.Errorf("trust cert: %w", err)
	}

	err = os.MkdirAll(coderdCertDir, 0666)
	if err != nil {
		return xerrors.Errorf("mkdir %s: %w", coderdCertDir, err)
	}

	certPath := filepath.Join(coderdCertDir, "certs.pem")
	file, err := os.OpenFile(certPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		return xerrors.Errorf("create file %s: %w", certPath, err)
	}

	for _, cert := range certs {
		_, _ = fmt.Fprintln(file, string(cert))
	}

	// Add our directory to the certs to trust
	certDir := os.Getenv("SSL_CERT_DIR")
	err = os.Setenv("SSL_CERT_DIR", certDir+":"+coderdCertDir)
	if err != nil {
		return xerrors.Errorf("set SSL_CERT_DIR: %w", err)
	}

	return nil
}
