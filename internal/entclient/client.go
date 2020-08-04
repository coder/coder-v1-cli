package entclient

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

func (c Client) copyURL() *url.URL {
	swp := *c.BaseURL
	return &swp
}

func (c *Client) http() (*http.Client, error) {
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
