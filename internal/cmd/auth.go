package cmd

import (
	"net/url"

	"golang.org/x/xerrors"

	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/coder-cli/internal/config"
)

var errNeedLogin = xerrors.New("failed to read session credentials: did you run \"coder login\"?")

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

	return &coder.Client{
		BaseURL: u,
		Token:   sessionToken,
	}, nil
}
