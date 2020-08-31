package coder

import (
	"context"
	"net/http"
	"time"

	"cdr.dev/coder-cli/internal/x/xjson"
	"nhooyr.io/websocket"
)

// Environment describes a Coder environment
type Environment struct {
	ID              string          `json:"id" tab:"-"`
	Name            string          `json:"name"`
	ImageID         string          `json:"image_id" tab:"-"`
	ImageTag        string          `json:"image_tag"`
	OrganizationID  string          `json:"organization_id" tab:"-"`
	UserID          string          `json:"user_id" tab:"-"`
	LastBuiltAt     time.Time       `json:"last_built_at" tab:"-"`
	CPUCores        float32         `json:"cpu_cores"`
	MemoryGB        int             `json:"memory_gb"`
	DiskGB          int             `json:"disk_gb"`
	GPUs            int             `json:"gpus"`
	Updating        bool            `json:"updating"`
	LatestStat      EnvironmentStat `json:"latest_stat" tab:"Status"`
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

// EnvironmentStat represents the state of an environment
type EnvironmentStat struct {
	Time            time.Time         `json:"time"`
	LastOnline      time.Time         `json:"last_online"`
	ContainerStatus EnvironmentStatus `json:"container_status"`
	StatError       string            `json:"stat_error"`
	CPUUsage        float32           `json:"cpu_usage"`
	MemoryTotal     int64             `json:"memory_total"`
	MemoryUsage     float32           `json:"memory_usage"`
	DiskTotal       int64             `json:"disk_total"`
	DiskUsed        int64             `json:"disk_used"`
}

func (e EnvironmentStat) String() string {
	return string(e.ContainerStatus)
}

// EnvironmentStatus refers to the states of an environment.
type EnvironmentStatus string

// The following represent the possible environment container states
const (
	EnvironmentCreating EnvironmentStatus = "CREATING"
	EnvironmentOff      EnvironmentStatus = "OFF"
	EnvironmentOn       EnvironmentStatus = "ON"
	EnvironmentFailed   EnvironmentStatus = "FAILED"
	EnvironmentUnknown  EnvironmentStatus = "UNKNOWN"
)

// CreateEnvironmentRequest is used to configure a new environment
type CreateEnvironmentRequest struct {
	Name     string   `json:"name"`
	ImageID  string   `json:"image_id"`
	ImageTag string   `json:"image_tag"`
	CPUCores float32  `json:"cpu_cores"`
	MemoryGB int      `json:"memory_gb"`
	DiskGB   int      `json:"disk_gb"`
	GPUs     int      `json:"gpus"`
	Services []string `json:"services"`
}

// CreateEnvironment sends a request to create an environment.
func (c Client) CreateEnvironment(ctx context.Context, orgID string, req CreateEnvironmentRequest) (*Environment, error) {
	var env *Environment
	err := c.requestBody(
		ctx,
		http.MethodPost, "/api/orgs/"+orgID+"/environments",
		req,
		env,
	)
	return env, err
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

// DeleteEnvironment deletes the environment.
func (c Client) DeleteEnvironment(ctx context.Context, envID string) error {
	return c.requestBody(
		ctx,
		http.MethodDelete, "/api/environments/"+envID,
		nil,
		nil,
	)
}

// DialWsep dials an environments command execution interface
// See github.com/cdr/wsep for details
func (c Client) DialWsep(ctx context.Context, env *Environment) (*websocket.Conn, error) {
	return c.dialWs(ctx, "/proxy/environments/"+env.ID+"/wsep")
}

// DialEnvironmentBuildLog opens a websocket connection for the environment build log messages
func (c Client) DialEnvironmentBuildLog(ctx context.Context, envID string) (*websocket.Conn, error) {
	return c.dialWs(ctx, "/api/environments/"+envID+"/watch-update")
}

// DialEnvironmentStats opens a websocket connection for environment stats
func (c Client) DialEnvironmentStats(ctx context.Context, envID string) (*websocket.Conn, error) {
	return c.dialWs(ctx, "/api/environments/"+envID+"/watch-stats")
}
