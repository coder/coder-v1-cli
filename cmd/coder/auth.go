package main

import (
	"net/url"

	"go.coder.com/flog"

	"cdr.dev/coder-cli/internal/entclient"
	"cdr.dev/coder-cli/internal/config"
)

func requireAuth() *entclient.Client {
	sessionToken, err := config.Session.Read()
	if err != nil {
		flog.Fatal("read session: %v (did you run coder login?)", err)
	}

	rawURL, err := config.URL.Read()
	if err != nil {
		flog.Fatal("read url: %v (did you run coder login?)", err)
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		flog.Fatal("url misformatted: %v (try runing coder login)", err)
	}

	return &entclient.Client{
		BaseURL: u,
		Token:   sessionToken,
	}
}
