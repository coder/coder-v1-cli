package cmd

import (
	"net/url"

	"golang.org/x/xerrors"

	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/coder-cli/internal/config"

	"go.coder.com/flog"
)

// requireAuth exits the process with a nonzero exit code if the user is not authenticated to make requests.
func requireAuth() *coder.Client {
	client, err := newClient()
	if err != nil {
		flog.Fatal("%s", err)
	}
	return client
}

func newClient() (*coder.Client, error) {
	sessionToken, err := config.Session.Read()
	if err != nil {
		return nil, xerrors.Errorf("read session: %w (did you run coder login?)", err)
	}

	rawURL, err := config.URL.Read()
	if err != nil {
		return nil, xerrors.Errorf("read url: %w (did you run coder login?)", err)
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, xerrors.Errorf("url misformatted: %w (try runing coder login)", err)
	}

	return &coder.Client{
		BaseURL: u,
		Token:   sessionToken,
	}, nil
}
