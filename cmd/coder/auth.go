package main

import (
	"net/url"

	"cdr.dev/coder-cli/internal/config"
	"cdr.dev/coder-cli/internal/entclient"

	"go.coder.com/flog"
)

// requireAuth exits the process with a nonzero exit code if the user is not authenticated to make requests
func requireAuth() *entclient.Client {
	sessionToken, err := config.Session.Read()
	requireSuccess(err, "read session: %v (did you run coder login?)", err)

	rawURL, err := config.URL.Read()
	requireSuccess(err, "read url: %v (did you run coder login?)", err)

	u, err := url.Parse(rawURL)
	requireSuccess(err, "url misformatted: %v (try runing coder login)", err)

	return &entclient.Client{
		BaseURL: u,
		Token:   sessionToken,
	}
}

// requireSuccess prints the given message and format args as a fatal error if err != nil
func requireSuccess(err error, msg string, args ...interface{}) {
	if err != nil {
		flog.Fatal(msg, args...)
	}
}
