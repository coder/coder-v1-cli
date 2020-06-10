package entclient

import (
	"context"
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
	if c.BaseURL.Scheme == "https" {
		u.Scheme = "wss"
	} else {
		u.Scheme = "ws"
	}
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
