package coder

import (
	"context"
	"net/http"
	"net/url"

	"nhooyr.io/websocket"
)

type requestOptions struct {
	BaseURLOverride *url.URL
	Query           url.Values
}

type requestOption func(*requestOptions)

// withQueryParams sets the provided query parameters on the request.
func withQueryParams(q url.Values) func(o *requestOptions) {
	return func(o *requestOptions) {
		o.Query = q
	}
}

func withBaseURL(base *url.URL) func(o *requestOptions) {
	return func(o *requestOptions) {
		o.BaseURLOverride = base
	}
}

// dialWebsocket establish the websocket connection while setting the authentication header.
func (c Client) dialWebsocket(ctx context.Context, path string, options ...requestOption) (*websocket.Conn, error) {
	// Make a copy of the url so we can update the scheme to ws(s) without mutating the state.
	url := *c.BaseURL
	var config requestOptions
	for _, o := range options {
		o(&config)
	}
	if config.BaseURLOverride != nil {
		url = *config.BaseURLOverride
	}
	url.Path = path

	conn, resp, err := websocket.Dial(ctx, url.String(), &websocket.DialOptions{HTTPHeader: http.Header{"Session-Token": {c.Token}}})
	if err != nil {
		if resp != nil {
			return nil, bodyError(resp)
		}
		return nil, err
	}

	return conn, nil
}
