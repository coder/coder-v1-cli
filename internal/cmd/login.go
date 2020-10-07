package cmd

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/coder-cli/internal/config"
	"cdr.dev/coder-cli/internal/loginsrv"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"golang.org/x/xerrors"

	"go.coder.com/flog"
)

func makeLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login [Coder Enterprise URL eg. https://my.coder.domain/]",
		Short: "Authenticate this client for future operations",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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
			u.Path = strings.TrimSuffix(u.Path, "/")

			// From this point, the commandline is correct.
			// Don't return errors as it would print the usage.

			if err := login(cmd, u, config.URL, config.Session); err != nil {
				return xerrors.Errorf("Login error", err)
			}
			return nil
		},
	}
}

// newLocalListener creates up a local tcp server using port 0 (i.e. any available port).
// If ipv4 is disabled, try ipv6.
// It will be used by the http server waiting for the auth callback.
func newLocalListener() (net.Listener, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		if l, err = net.Listen("tcp6", "[::1]:0"); err != nil {
			return nil, xerrors.Errorf("listen on a port: %w", err)
		}
	}
	return l, nil
}

// pingAPI creates a client from the given url/token and try to exec an api call.
// Not using the SDK as we want to verify the url/token pair before storing the config files.
func pingAPI(ctx context.Context, envURL *url.URL, token string) error {
	client := &coder.Client{BaseURL: envURL, Token: token}
	if _, err := client.Me(ctx); err != nil {
		return xerrors.Errorf("call api: %w", err)
	}
	return nil
}

// storeConfig writes the env URL and session token to the local config directory.
// The config lib will handle the local config path lookup and creation.
func storeConfig(envURL *url.URL, sessionToken string, urlCfg, sessionCfg config.File) error {
	if err := urlCfg.Write(envURL.String()); err != nil {
		return xerrors.Errorf("store env url: %w", err)
	}
	if err := sessionCfg.Write(sessionToken); err != nil {
		return xerrors.Errorf("store session token: %w", err)
	}
	return nil
}

func login(cmd *cobra.Command, envURL *url.URL, urlCfg, sessionCfg config.File) error {
	ctx := cmd.Context()

	// Start by creating the listener so we can prompt the user with the URL.
	listener, err := newLocalListener()
	if err != nil {
		return xerrors.Errorf("create local listener: %w", err)
	}
	defer func() { _ = listener.Close() }() // Best effort.

	// Forge the auth URL with the callback set to the local server.
	authURL := *envURL
	authURL.Path = envURL.Path + "/internal-auth"
	authURL.RawQuery = "local_service=http://" + listener.Addr().String()

	// Try to open the browser on the local computer.
	if err := browser.OpenURL(authURL.String()); err != nil {
		// Discard the error as it is an expected one in non-X environments like over ssh.
		// Tell the user to visit the URL instead.
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Visit the following URL in your browser:\n\n\t%s\n\n", &authURL) // Can't fail.
	}

	// Create our channel, it is going to be the central synchronization of the command.
	tokenChan := make(chan string)

	// Create the http server outside the errgroup goroutine scope so we can stop it later.
	srv := &http.Server{Handler: &loginsrv.Server{TokenChan: tokenChan}}
	defer func() { _ = srv.Close() }() // Best effort. Direct close as we are dealing with a one-off request.

	// Start both the readline and http server in parallel. As they are both long-running routines,
	// to know when to continue, we don't wait on the errgroup, but on the tokenChan.
	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error { return srv.Serve(listener) })
	group.Go(func() error { return loginsrv.ReadLine(ctx, cmd.InOrStdin(), cmd.ErrOrStderr(), tokenChan) })

	// Only close then tokenChan when the errgroup is done. Best effort basis.
	// Will not return the http route is used with a regular terminal.
	// Useful for non interactive session, manual input, tests or custom stdin.
	go func() { defer close(tokenChan); _ = group.Wait() }()

	var token string
	select {
	case <-ctx.Done():
		return ctx.Err()
	case token = <-tokenChan:
	}

	// Perform an API call to verify that the token is valid.
	if err := pingAPI(ctx, envURL, token); err != nil {
		return xerrors.Errorf("ping API: %w", err)
	}

	// Success. Store the config only at this point so we don't override the local one in case of failure.
	if err := storeConfig(envURL, token, urlCfg, sessionCfg); err != nil {
		return xerrors.Errorf("store config: %w", err)
	}

	flog.Success("Logged in.")

	return nil
}
