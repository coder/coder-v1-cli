package coder

import (
	"context"
	"net/http"

	"nhooyr.io/websocket"
)

// dialWebsocket establish the websocket connection while setting the authentication header.
func (c Client) dialWebsocket(ctx context.Context, path string) (*websocket.Conn, error) {
	// Make a copy of the url so we can update the scheme to ws(s) without mutating the state.
	url := *c.BaseURL
	if url.Scheme == "https" {
		url.Scheme = "wss"
	} else {
		url.Scheme = "ws"
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
