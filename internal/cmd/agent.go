package cmd

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	// We use slog here since agent runs in the background and we can benefit
	// from structured logging.
	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/sloghuman"
	"github.com/spf13/cobra"
	"golang.org/x/xerrors"

	"cdr.dev/coder-cli/wsnet"
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
		logFile  string
	)
	cmd := &cobra.Command{
		Use:   "start --coder-url=<coder_url> --token=<token> --log-file=<path>",
		Short: "starts the coder agent",
		Long:  "starts the coder agent",
		Example: `# start the agent and use CODER_URL and CODER_AGENT_TOKEN env vars
coder agent start

# start the agent and connect with a specified url and agent token
coder agent start --coder-url https://my-coder.com --token xxxx-xxxx

# start the agent and write a copy of the log to /tmp/coder-agent.log
# if the file already exists, it will be truncated
coder agent start --log-file=/tmp/coder-agent.log
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			log := slog.Make(sloghuman.Sink(os.Stderr)).Leveled(slog.LevelDebug)

			// Optional log file path to write
			if logFile != "" {
				// Truncate the file if it already exists
				file, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
				if err != nil {
					// If an error occurs, log it as an error, but consider it non-fatal
					log.Warn(ctx, "failed to open log file", slog.Error(err))
				} else {
					// Log to both standard output and our file
					log = slog.Make(
						sloghuman.Sink(os.Stderr),
						sloghuman.Sink(file),
					).Leveled(slog.LevelDebug)
				}
			}

			if coderURL == "" {
				var ok bool
				coderURL, ok = os.LookupEnv("CODER_URL")
				if !ok {
					client, err := newClient(ctx, true)
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

			// First inject certs
			err = writeCoderdCerts(ctx)
			if err != nil {
				return xerrors.Errorf("trust certs: %w", err)
			}

			log.Info(ctx, "starting wsnet listener", slog.F("coder_access_url", u.String()))
			listener, err := wsnet.Listen(ctx, log, wsnet.ListenEndpoint(u, token), token)
			if err != nil {
				return xerrors.Errorf("listen: %w", err)
			}
			defer func() {
				log.Info(ctx, "closing wsnet listener")
				err := listener.Close()
				if err != nil {
					log.Error(ctx, "close listener", slog.Error(err))
				}
			}()

			// Block until user sends SIGINT or SIGTERM
			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
			<-sigs

			return nil
		},
	}

	cmd.Flags().StringVar(&token, "token", "", "coder agent token")
	cmd.Flags().StringVar(&coderURL, "coder-url", "", "coder access url")
	cmd.Flags().StringVar(&logFile, "log-file", "", "write a copy of logs to file")

	return cmd
}

func writeCoderdCerts(ctx context.Context) error {
	// Inject certs to custom dir and concat with : with existing dir.
	certs, err := trustCertificate(ctx)
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

// trustCertificate will fetch coderd's certificate and write it to disc.
// It will then extend the certs to trust to include this directory.
// This only happens if coderd can answer the challenge to prove
// it has the shared secret.
func trustCertificate(ctx context.Context) ([][]byte, error) {
	conf := &tls.Config{InsecureSkipVerify: true}
	hc := &http.Client{
		Timeout: time.Second * 3,
		Transport: &http.Transport{
			TLSClientConfig: conf,
		},
	}

	c, err := newClient(ctx, true, withHTTPClient(hc))
	if err != nil {
		return nil, xerrors.Errorf("new client: %w", err)
	}

	id := os.Getenv("CODER_WORKSPACE_ID")
	challenge, err := c.TrustEnvironment(ctx, id)
	if err != nil {
		return nil, xerrors.Errorf("challenge failed: %w", err)
	}

	return challenge.Certificates, nil
}
