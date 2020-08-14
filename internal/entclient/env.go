package entclient

import (
	"context"
	"net/http"
	"time"

	"cdr.dev/coder-cli/internal/x/xjson"
	"nhooyr.io/websocket"
)

// Environment describes a Coder environment
type Environment struct {
	ID              string    `json:"id" tab:"-"`
	Name            string    `json:"name"`
	ImageID         string    `json:"image_id" tab:"-"`
	ImageTag        string    `json:"image_tag"`
	OrganizationID  string    `json:"organization_id" tab:"-"`
	UserID          string    `json:"user_id" tab:"-"`
	LastBuiltAt     time.Time `json:"last_built_at" tab:"-"`
	CPUCores        float32   `json:"cpu_cores"`
	MemoryGB        int       `json:"memory_gb"`
	DiskGB          int       `json:"disk_gb"`
	GPUs            int       `json:"gpus"`
	Updating        bool      `json:"updating"`
	RebuildMessages []struct {
		Text     string `json:"text"`
		Required bool   `json:"required"`
	} `json:"rebuild_messages" tab:"-"`
	CreatedAt        time.Time      `json:"created_at" tab:"-"`
	UpdatedAt        time.Time      `json:"updated_at" tab:"-"`
	LastOpenedAt     time.Time      `json:"last_opened_at" tab:"-"`
	LastConnectionAt time.Time      `json:"last_connection_at" tab:"-"`
	AutoOffThreshold xjson.Duration `json:"auto_off_threshold" tab:"-"`
}

// EnvironmentsInOrganization gets the list of environments owned by the authenticated user
func (c Client) EnvironmentsInOrganization(ctx context.Context, user *User, org *Org) ([]Environment, error) {
	var envs []Environment
	err := c.requestBody(
		ctx,
		http.MethodGet, "/api/orgs/"+org.ID+"/members/"+user.ID+"/environments",
		nil,
		&envs,
	)
	return envs, err
}

// DialWsep dials an environments command execution interface
// See github.com/cdr/wsep for details
func (c Client) DialWsep(ctx context.Context, env *Environment) (*websocket.Conn, error) {
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
