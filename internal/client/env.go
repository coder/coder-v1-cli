package client

import (
	"context"
	"net/url"
	"strconv"
	"time"

	"nhooyr.io/websocket"
)

type Environment struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

func (c Client) Envs(user *User, org Org) ([]Environment, error) {
	var envs []Environment
	err := c.requestBody(
		"GET", "/api/environments/?user_id="+user.ID+"&organization_id="+org.ID,
		nil,
		&envs,
	)
	return envs, err
}

type WushOptions struct {
	TTY   bool
	Stdin bool
}

var defaultWushOptions = WushOptions{
	TTY:   false,
	Stdin: true,
}

func (c Client) DialWush(env Environment, opts *WushOptions, cmd string, args ...string) (*websocket.Conn, error) {
	u := c.copyURL()
	if c.BaseURL.Scheme == "https" {
		u.Scheme = "wss"
	} else {
		u.Scheme = "ws"
	}
	u.Path = "/proxy/environments/" + env.ID + "/wush-lite"
	query := make(url.Values)
	query.Set("command", cmd)
	query["args[]"] = args
	if opts == nil {
		opts = &defaultWushOptions
	}
	query.Set("tty", strconv.FormatBool(opts.TTY))
	query.Set("stdin", strconv.FormatBool(opts.Stdin))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	fullURL := u.String() + "?" + query.Encode()

	conn, resp, err := websocket.Dial(ctx, fullURL,
		&websocket.DialOptions{
			HTTPHeader: map[string][]string{
				"Cookie": {"session_token=" + c.Token},
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
