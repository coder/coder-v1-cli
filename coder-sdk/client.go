package coder

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
)

// Me is the route param to access resources of the authenticated user
const Me = "me"

// Client wraps the Coder HTTP API
type Client struct {
	BaseURL *url.URL
	Token   string
}

// newHTTPClient creates a default underlying http client and sets the auth cookie.
//
// NOTE: As we do not specify a custom transport, the default one from the stdlib will be used,
//       resulting in a persistent connection pool.
//       We do not set a timeout here as it could cause issue with the websocket.
//       The caller is expected to set it when needed.
//
// WARNING: If the caller sets a custom transport to set TLS settings or a custom CA, the default
//          pool will not be used and it might result in a new dns lookup/tls session/socket begin
//          established each time.
func (c *Client) newHTTPClient() (*http.Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	jar.SetCookies(c.BaseURL, []*http.Cookie{
		{
			Name:     "session_token",
			Value:    c.Token,
			MaxAge:   86400,
			Path:     "/",
			HttpOnly: true,
			Secure:   c.BaseURL.Scheme == "https",
		},
	})

	return &http.Client{Jar: jar}, nil
}
