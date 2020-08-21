package cmd

import (
	"net/url"

	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/coder-cli/internal/config"
	"golang.org/x/xerrors"

	"go.coder.com/flog"
)

// RequireAuth exits the process with a nonzero exit code if the user is not authenticated to make requests
func RequireAuth() *coder.Client {
	client, err := newClient()
	if err != nil {
		flog.Fatal("%v", err)
	}
	return client
}

func newClient() (*coder.Client, error) {
	sessionToken, err := config.Session.Read()
	if err != nil {
		return nil, xerrors.Errorf("read session: %v (did you run coder login?)", err)
	}

	rawURL, err := config.URL.Read()
	if err != nil {
		return nil, xerrors.Errorf("read url: %v (did you run coder login?)", err)
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, xerrors.Errorf("url misformatted: %v (try runing coder login)", err)
	}

	client := &coder.Client{
		BaseURL: u,
		Token:   sessionToken,
	}

	return client, nil
}
