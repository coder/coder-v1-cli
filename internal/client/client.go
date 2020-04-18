package client

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
)

type Client struct {
	BaseURL *url.URL
	Token   string
}

func (c Client) copyURL() *url.URL {
	return &(*c.BaseURL)
}

func (c *Client) httpClient() (*http.Client, error) {
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
