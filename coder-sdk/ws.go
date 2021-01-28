package coder

import (
	"context"
	"net/http"

	"nhooyr.io/websocket"
)

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
