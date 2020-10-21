package cmd

import (
	"context"
	"net/http"
	"net/url"

	"golang.org/x/xerrors"

	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/coder-cli/internal/clog"
	"cdr.dev/coder-cli/internal/config"
)

var errNeedLogin = clog.Fatal(
	"failed to read session credentials",
	clog.Hint(`did you run "coder login [https://coder.domain.com]"?`),
)

func newClient() (*coder.Client, error) {
	sessionToken, err := config.Session.Read()
	if err != nil {
		return nil, errNeedLogin
	}

	rawURL, err := config.URL.Read()
	if err != nil {
		return nil, errNeedLogin
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, xerrors.Errorf("url misformatted: %w try runing \"coder login\" with a valid URL", err)
	}

	c := &coder.Client{
		BaseURL: u,
		Token:   sessionToken,
	}

	// Make sure we can make a request so the final
	// error is more clean.
	_, err = c.Me(context.Background())
	if err != nil {
		var he *coder.HTTPError
		if xerrors.As(err, &he) {
			switch he.StatusCode {
			case http.StatusUnauthorized:
				return nil, xerrors.Errorf("not authenticated: try running \"coder login`\"")
			}
		}
		return nil, err
	}

	return c, nil
}
