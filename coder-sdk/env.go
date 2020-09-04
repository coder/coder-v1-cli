package coder

import (
	"context"
	"net/http"
	"time"

	"cdr.dev/coder-cli/internal/x/xjson"
	"golang.org/x/xerrors"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

// Environment describes a Coder environment
type Environment struct {
	ID               string           `json:"id"                 tab:"-"`
	Name             string           `json:"name"               tab:"Name"`
	ImageID          string           `json:"image_id"           tab:"-"`
	ImageTag         string           `json:"image_tag"          tab:"ImageTag"`
	OrganizationID   string           `json:"organization_id"    tab:"-"`
	UserID           string           `json:"user_id"            tab:"-"`
	LastBuiltAt      time.Time        `json:"last_built_at"      tab:"-"`
	CPUCores         float32          `json:"cpu_cores"          tab:"CPUCores"`
	MemoryGB         int              `json:"memory_gb"          tab:"MemoryGB"`
	DiskGB           int              `json:"disk_gb"            tab:"DiskGB"`
	GPUs             int              `json:"gpus"               tab:"GPUs"`
	Updating         bool             `json:"updating"           tab:"Updating"`
	LatestStat       EnvironmentStat  `json:"latest_stat"        tab:"Status"`
	RebuildMessages  []RebuildMessage `json:"rebuild_messages"   tab:"-"`
	CreatedAt        time.Time        `json:"created_at"         tab:"-"`
	UpdatedAt        time.Time        `json:"updated_at"         tab:"-"`
	LastOpenedAt     time.Time        `json:"last_opened_at"     tab:"-"`
	LastConnectionAt time.Time        `json:"last_connection_at" tab:"-"`
	AutoOffThreshold xjson.Duration   `json:"auto_off_threshold" tab:"-"`
}

// RebuildMessage defines the message shown when an Environment requires a rebuild for it can be accessed.
type RebuildMessage struct {
	Text     string `json:"text"`
	Required bool   `json:"required"`
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
	var env Environment
	err := c.requestBody(
		ctx,
		http.MethodPost, "/api/orgs/"+orgID+"/environments",
		req,
		&env,
	)
	return &env, err
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

// DialIDEStatus opens a websocket connection for cpu load metrics on the environment
func (c Client) DialIDEStatus(ctx context.Context, envID string) (*websocket.Conn, error) {
	return c.dialWs(ctx, "/proxy/environments/"+envID+"/ide/api/status")
}

// DialEnvironmentBuildLog opens a websocket connection for the environment build log messages
func (c Client) DialEnvironmentBuildLog(ctx context.Context, envID string) (*websocket.Conn, error) {
	return c.dialWs(ctx, "/api/environments/"+envID+"/watch-update")
}

// DialEnvironmentStats opens a websocket connection for environment stats
func (c Client) DialEnvironmentStats(ctx context.Context, envID string) (*websocket.Conn, error) {
	return c.dialWs(ctx, "/api/environments/"+envID+"/watch-stats")
}

// DialResourceLoad opens a websocket connection for cpu load metrics on the environment
func (c Client) DialResourceLoad(ctx context.Context, envID string) (*websocket.Conn, error) {
	return c.dialWs(ctx, "/api/environments/"+envID+"/watch-resource-load")
}

// BuildLogType describes the type of an event.
type BuildLogType string

const (
	// BuildLogTypeStart signals that a new build log has begun.
	BuildLogTypeStart BuildLogType = "start"
	// BuildLogTypeStage is a stage-level event for an environment.
	// It can be thought of as a major step in the environment's
	// lifecycle.
	BuildLogTypeStage BuildLogType = "stage"
	// BuildLogTypeError describes an error that has occurred.
	BuildLogTypeError BuildLogType = "error"
	// BuildLogTypeSubstage describes a subevent that occurs as
	// part of a stage. This can be the output from a user's
	// personalization script, or a long running command.
	BuildLogTypeSubstage BuildLogType = "substage"
	// BuildLogTypeDone signals that the build has completed.
	BuildLogTypeDone BuildLogType = "done"
)

type buildLogMsg struct {
	Type BuildLogType `json:"type"`
}

// WaitForEnvironmentReady will watch the build log and return when done
func (c Client) WaitForEnvironmentReady(ctx context.Context, env *Environment) error {
	conn, err := c.DialEnvironmentBuildLog(ctx, env.ID)
	if err != nil {
		return xerrors.Errorf("%s: dial build log: %w", env.Name, err)
	}

	for {
		msg := buildLogMsg{}
		err := wsjson.Read(ctx, conn, &msg)
		if err != nil {
			return xerrors.Errorf("%s: reading build log msg: %w", env.Name, err)
		}

		if msg.Type == BuildLogTypeDone {
			return nil
		}
	}
}
