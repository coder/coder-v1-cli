package coder

import (
	"context"

	"nhooyr.io/websocket"
)

func (c Client) dialWs(ctx context.Context, path string) (*websocket.Conn, error) {
	u := c.copyURL()
	if c.BaseURL.Scheme == "https" {
		u.Scheme = "wss"
	} else {
		u.Scheme = "ws"
	}
	u.Path = path

	conn, resp, err := websocket.Dial(ctx, u.String(),
		&websocket.DialOptions{
			HTTPHeader: map[string][]string{
				"Session-Token": {c.Token},
			},
		},
	)
	if err != nil {
		if resp != nil {
			return nil, bodyError(resp)
		}
		return nil, err
	}

	return conn, nil
}
