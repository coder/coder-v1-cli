package entclient

import (
	"context"
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
		"GET", "/api/orgs/"+org.ID+"/members/"+user.ID+"/environments",
		nil,
		&envs,
	)
	return envs, err
}

func (c Client) DialWsep(ctx context.Context, env Environment) (*websocket.Conn, error) {
	u := c.copyURL()
	u.Scheme = c.wsScheme()
	u.Path = "/proxy/environments/" + env.ID + "/wsep"

	ctx, cancel := context.WithTimeout(ctx, time.Second*15)
	defer cancel()

	conn, resp, err := websocket.Dial(ctx, u.String(),
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

type LogOptions struct {
	Container string
	Follow    bool
}

func (c Client) Logs(ctx context.Context, env Environment, opts LogOptions) (*websocket.Conn, error) {
	u := c.copyURL()
	u.Scheme = c.wsScheme()
	u.Path = "/api/environments/" + env.ID + "/watch-logs"

	vals := u.Query()
	vals.Set("container", opts.Container)
	vals.Set("follow", strconv.FormatBool(opts.Follow))
	u.RawQuery = vals.Encode()

	ctx, cancel := context.WithTimeout(ctx, time.Second*15)
	defer cancel()

	conn, resp, err := websocket.Dial(ctx, u.String(),
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
