package cmd

import (
	"bufio"
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"golang.org/x/xerrors"

	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/coder-cli/internal/config"
	"cdr.dev/coder-cli/internal/version"
	"cdr.dev/coder-cli/internal/x/xcobra"
	"cdr.dev/coder-cli/pkg/clog"
)

func loginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login [Coder URL eg. https://my.coder.domain/]",
		Short: "Authenticate this client for future operations",
		Args:  xcobra.ExactArgs(1),
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

			if err := login(cmd, u); err != nil {
				return xerrors.Errorf("login error: %w", err)
			}
			return nil
		},
	}
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

func login(cmd *cobra.Command, envURL *url.URL) error {
	authURL := *envURL
	authURL.Path = envURL.Path + "/internal-auth"
	q := authURL.Query()
	q.Add("show_token", "true")
	authURL.RawQuery = q.Encode()

	if err := browser.OpenURL(authURL.String()); err != nil {
		fmt.Printf("Open the following in your browser:\n\n\t%s\n\n", authURL.String())
	} else {
		fmt.Printf("Your browser has been opened to visit:\n\n\t%s\n\n", authURL.String())
	}

	fmt.Print("Paste token here: ")
	var token string
	scanner := bufio.NewScanner(cmd.InOrStdin())
	scanner.Scan()
	token = scanner.Text()
	if err := scanner.Err(); err != nil {
		return xerrors.Errorf("reading standard input: %w", err)
	}

	if err := pingAPI(cmd.Context(), envURL, token); err != nil {
		return xerrors.Errorf("ping API with credentials: %w", err)
	}
	if err := storeConfig(envURL, token, config.URL, config.Session); err != nil {
		return xerrors.Errorf("store auth: %w", err)
	}
	clog.LogSuccess("logged in")
	return nil
}

// pingAPI creates a client from the given url/token and try to exec an api call.
// Not using the SDK as we want to verify the url/token pair before storing the config files.
func pingAPI(ctx context.Context, envURL *url.URL, token string) error {
	client, err := coder.NewClient(coder.ClientOptions{
		BaseURL: envURL,
		Token:   token,
	})
	if err != nil {
		return xerrors.Errorf("failed to create coder.Client: %w", err)
	}

	if apiVersion, err := client.APIVersion(ctx); err == nil {
		if apiVersion != "" && !version.VersionsMatch(apiVersion) {
			logVersionMismatchError(apiVersion)
		}
	}
	_, err = client.Me(ctx)
	if err != nil {
		return xerrors.Errorf("call api: %w", err)
	}
	return nil
}
