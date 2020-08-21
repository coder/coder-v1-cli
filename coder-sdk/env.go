package coder

import (
	"context"
	"fmt"
	"net/http"
	"nhooyr.io/websocket/wsjson"
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

// EnvironmentsByOrganization gets the list of environments owned by the given user.
func (c Client) EnvironmentsByOrganization(ctx context.Context, userID, orgID string) ([]Environment, error) {
	var envs []Environment
	err := c.requestBody(
		ctx,
		http.MethodGet, "/api/orgs/"+orgID+"/members/"+userID+"/environments",
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

type CreateEnvironmentRequest struct {
	Name     string   `json:"name"`
	ImageID  string   `json:"image_id"`
	ImageTag string   `json:"image_tag"`
	CPUCores float32  `json:"cpu_cores"`
	MemoryGB int      `json:"memory_gb"`
	DiskGB   int      `json:"disk_gb"`
	GPUs     int      `json:"gpus"` // can be set to 0
	Services []string `json:"services"`
}

func (c Client) CreateEnvironment(ctx context.Context, orgID string, req CreateEnvironmentRequest) (Environment, error) {
	var env Environment
	err := c.requestBody(
		ctx,
		http.MethodPost, "/api/orgs/"+orgID+"/environments",
		req,
		&env,
	)
	return env, err
}

type envUpdate struct {
	Type string `json:"type"`
}

func (c Client) WaitForEnvironmentReady(ctx context.Context, envID string) error {
	u := c.copyURL()
	if c.BaseURL.Scheme == "https" {
		u.Scheme = "wss"
	} else {
		u.Scheme = "ws"
	}
	u.Path = "/api/environments/" + envID + "/watch-update"

	conn, resp, err := websocket.Dial(ctx, u.String(),
		&websocket.DialOptions{
			HTTPHeader: map[string][]string{
				"Cookie": {"session_token=" + c.Token},
			},
		},
	)
	if err != nil {
		if resp != nil {
			return bodyError(resp)
		}
		return err
	}

	for {
		m := envUpdate{}
		err = wsjson.Read(ctx, conn, &m)
		if err != nil {
			return fmt.Errorf("read ws json msg: %w", err)
		}
		if m.Type == "done" {
			break
		}
	}

	return nil
}

type stats struct {
	ContainerStatus string `json:"container_status"`
	StatError       string `json:"stat_error"`
	Time            string `json:"time"`
}

func (c Client) WatchEnvironmentStats(ctx context.Context, envID string, duration time.Duration) error {
	u := c.copyURL()
	if c.BaseURL.Scheme == "https" {
		u.Scheme = "wss"
	} else {
		u.Scheme = "ws"
	}
	u.Path = "/api/environments/" + envID + "/watch-stats"

	conn, resp, err := websocket.Dial(ctx, u.String(),
		&websocket.DialOptions{
			HTTPHeader: map[string][]string{
				"Cookie": {"session_token=" + c.Token},
			},
		},
	)
	if err != nil {
		if resp != nil {
			return bodyError(resp)
		}
		return err
	}

	statsCtx, statsCancel := context.WithTimeout(ctx, duration)
	defer statsCancel()

	for {
		select {
		case <-statsCtx.Done():
			return nil
		default:
			m := stats{}
			err = wsjson.Read(ctx, conn, &m)
			if err != nil {
				return fmt.Errorf("read ws json msg: %w", err)
			}
		}
	}
}

func (c Client) DeleteEnvironment(ctx context.Context, envID string) error {
	err := c.requestBody(
		ctx,
		http.MethodDelete, "/api/environments/" + envID,
		nil,
		nil,
	)
	return err
}