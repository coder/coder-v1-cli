package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"golang.org/x/xerrors"

	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/coder-cli/internal/config"
	"cdr.dev/coder-cli/internal/version"
	"cdr.dev/coder-cli/pkg/clog"
)

var errNeedLogin = clog.Fatal(
	"failed to read session credentials",
	clog.Hintf(`did you run "coder login [https://coder.domain.com]"?`),
)

const tokenEnv = "CODER_TOKEN"
const urlEnv = "CODER_URL"

func newClient(ctx context.Context) (coder.Client, error) {
	var (
		err          error
		sessionToken = os.Getenv(tokenEnv)
		rawURL       = os.Getenv(urlEnv)
	)

	if sessionToken == "" || rawURL == "" {
		sessionToken, err = config.Session.Read()
		if err != nil {
			return nil, errNeedLogin
		}

		rawURL, err = config.URL.Read()
		if err != nil {
			return nil, errNeedLogin
		}
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, xerrors.Errorf("url malformed: %w try running \"coder login\" with a valid URL", err)
	}

	c, err := coder.NewClient(coder.ClientOptions{
		BaseURL: u,
		Token:   sessionToken,
	})
	if err != nil {
		return nil, xerrors.Errorf("failed to create new coder.Client: %w", err)
	}

	apiVersion, err := c.APIVersion(ctx)
	if apiVersion != "" && !version.VersionsMatch(apiVersion) {
		logVersionMismatchError(apiVersion)
	}
	if err != nil {
		var he *coder.HTTPError
		if xerrors.As(err, &he) {
			if he.StatusCode == http.StatusUnauthorized {
				return nil, xerrors.Errorf("not authenticated: try running \"coder login`\"")
			}
		}
		return nil, err
	}

	return c, nil
}

func logVersionMismatchError(apiVersion string) {
	clog.LogWarn(
		"version mismatch detected",
		fmt.Sprintf("Coder CLI version: %s", version.Version),
		fmt.Sprintf("Coder API version: %s", apiVersion), clog.BlankLine,
		clog.Tipf("download the appropriate version here: https://github.com/cdr/coder-cli/releases"),
	)
}
