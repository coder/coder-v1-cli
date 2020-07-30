package main

import (
	"net/url"

	"cdr.dev/coder-cli/internal/xcli"

	"cdr.dev/coder-cli/internal/config"
	"cdr.dev/coder-cli/internal/entclient"
)

func requireAuth() *entclient.Client {
	sessionToken, err := config.Session.Read()
	xcli.RequireSuccess(err, "read session: %v (did you run coder login?)", err)

	rawURL, err := config.URL.Read()
	xcli.RequireSuccess(err, "read url: %v (did you run coder login?)", err)

	u, err := url.Parse(rawURL)
	xcli.RequireSuccess(err, "url misformatted: %v (try runing coder login)", err)

	return &entclient.Client{
		BaseURL: u,
		Token:   sessionToken,
	}
}
