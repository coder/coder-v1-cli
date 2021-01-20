package coder

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"cdr.dev/wsep"
	"golang.org/x/xerrors"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

// Environment describes a Coder environment.
type Environment struct {
	ID               string           `json:"id"                 table:"-"`
	Name             string           `json:"name"               table:"Name"`
	ImageID          string           `json:"image_id"           table:"-"`
	ImageTag         string           `json:"image_tag"          table:"ImageTag"`
	OrganizationID   string           `json:"organization_id"    table:"-"`
	UserID           string           `json:"user_id"            table:"-"`
	LastBuiltAt      time.Time        `json:"last_built_at"      table:"-"`
	CPUCores         float32          `json:"cpu_cores"          table:"CPUCores"`
	MemoryGB         float32          `json:"memory_gb"          table:"MemoryGB"`
	DiskGB           int              `json:"disk_gb"            table:"DiskGB"`
	GPUs             int              `json:"gpus"               table:"GPUs"`
	Updating         bool             `json:"updating"           table:"Updating"`
	LatestStat       EnvironmentStat  `json:"latest_stat"        table:"Status"`
	RebuildMessages  []RebuildMessage `json:"rebuild_messages"   table:"-"`
	CreatedAt        time.Time        `json:"created_at"         table:"-"`
	UpdatedAt        time.Time        `json:"updated_at"         table:"-"`
	LastOpenedAt     time.Time        `json:"last_opened_at"     table:"-"`
	LastConnectionAt time.Time        `json:"last_connection_at" table:"-"`
	AutoOffThreshold Duration         `json:"auto_off_threshold" table:"-"`
	UseContainerVM   bool             `json:"use_container_vm"   table:"CVM"`
	ResourcePoolID   string           `json:"resource_pool_id"   table:"-"`
}

// RebuildMessage defines the message shown when an Environment requires a rebuild for it can be accessed.
type RebuildMessage struct {
	Text             string   `json:"text"`
	Required         bool     `json:"required"`
	AutoOffThreshold Duration `json:"auto_off_threshold"`
}

// EnvironmentStat represents the state of an environment.
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

func (e EnvironmentStat) String() string { return string(e.ContainerStatus) }

// EnvironmentStatus refers to the states of an environment.
type EnvironmentStatus string

// The following represent the possible environment container states.
const (
	EnvironmentCreating EnvironmentStatus = "CREATING"
	EnvironmentOff      EnvironmentStatus = "OFF"
	EnvironmentOn       EnvironmentStatus = "ON"
	EnvironmentFailed   EnvironmentStatus = "FAILED"
	EnvironmentUnknown  EnvironmentStatus = "UNKNOWN"
)

// CreateEnvironmentRequest is used to configure a new environment.
type CreateEnvironmentRequest struct {
	Name           string   `json:"name"`
	ImageID        string   `json:"image_id"`
	OrgID          string   `json:"org_id"`
	ImageTag       string   `json:"image_tag"`
	CPUCores       float32  `json:"cpu_cores"`
	MemoryGB       float32  `json:"memory_gb"`
	DiskGB         int      `json:"disk_gb"`
	GPUs           int      `json:"gpus"`
	Services       []string `json:"services"`
	UseContainerVM bool     `json:"use_container_vm"`
}

// CreateEnvironment sends a request to create an environment.
func (c Client) CreateEnvironment(ctx context.Context, req CreateEnvironmentRequest) (*Environment, error) {
	var env Environment
	if err := c.requestBody(ctx, http.MethodPost, "/api/v0/environments", req, &env); err != nil {
		return nil, err
	}
	return &env, nil
}

// Environments lists environments returned by the given filter.
// TODO: add the filter options, explore performance issue.
func (c Client) Environments(ctx context.Context) ([]Environment, error) {
	var envs []Environment
	if err := c.requestBody(ctx, http.MethodGet, "/api/private/environments", nil, &envs); err != nil {
		return nil, err
	}
	return envs, nil
}

// UserEnvironmentsByOrganization gets the list of environments owned by the given user.
func (c Client) UserEnvironmentsByOrganization(ctx context.Context, userID, orgID string) ([]Environment, error) {
	var (
		envs  []Environment
		query = url.Values{}
	)

	query.Add("orgs", orgID)
	query.Add("users", userID)

	if err := c.requestBody(ctx, http.MethodGet, "/api/v0/environments", nil, &envs, withQueryParams(query)); err != nil {
		return nil, err
	}
	return envs, nil
}

// DeleteEnvironment deletes the environment.
func (c Client) DeleteEnvironment(ctx context.Context, envID string) error {
	return c.requestBody(ctx, http.MethodDelete, "/api/private/environments/"+envID, nil, nil)
}

// StopEnvironment stops the stops.
func (c Client) StopEnvironment(ctx context.Context, envID string) error {
	return c.requestBody(ctx, http.MethodPut, "/api/private/environments/"+envID+"/stop", nil, nil)
}

// UpdateEnvironmentReq defines the update operation, only setting
// nil-fields.
type UpdateEnvironmentReq struct {
	ImageID              *string   `json:"image_id"`
	ImageTag             *string   `json:"image_tag"`
	CPUCores             *float32  `json:"cpu_cores"`
	MemoryGB             *float32  `json:"memory_gb"`
	DiskGB               *int      `json:"disk_gb"`
	GPUs                 *int      `json:"gpus"`
	Services             *[]string `json:"services"`
	CodeServerReleaseURL *string   `json:"code_server_release_url"`
}

// RebuildEnvironment requests that the given envID is rebuilt with no changes to its specification.
func (c Client) RebuildEnvironment(ctx context.Context, envID string) error {
	return c.requestBody(ctx, http.MethodPatch, "/api/private/environments/"+envID, UpdateEnvironmentReq{}, nil)
}

// EditEnvironment modifies the environment specification and initiates a rebuild.
func (c Client) EditEnvironment(ctx context.Context, envID string, req UpdateEnvironmentReq) error {
	return c.requestBody(ctx, http.MethodPatch, "/api/private/environments/"+envID, req, nil)
}

// DialWsep dials an environments command execution interface
// See https://github.com/cdr/wsep for details.
func (c Client) DialWsep(ctx context.Context, baseURL *url.URL, envID string) (*websocket.Conn, error) {
	return c.dialWebsocket(ctx, "/proxy/environments/"+envID+"/wsep", withBaseURL(baseURL))
}

// DialExecutor gives a remote execution interface for performing commands inside an environment.
func (c Client) DialExecutor(ctx context.Context, baseURL *url.URL, envID string) (wsep.Execer, error) {
	ws, err := c.DialWsep(ctx, baseURL, envID)
	if err != nil {
		return nil, err
	}
	return wsep.RemoteExecer(ws), nil
}

// DialIDEStatus opens a websocket connection for cpu load metrics on the environment.
func (c Client) DialIDEStatus(ctx context.Context, baseURL *url.URL, envID string) (*websocket.Conn, error) {
	return c.dialWebsocket(ctx, "/proxy/environments/"+envID+"/ide/api/status", withBaseURL(baseURL))
}

// DialEnvironmentBuildLog opens a websocket connection for the environment build log messages.
func (c Client) DialEnvironmentBuildLog(ctx context.Context, envID string) (*websocket.Conn, error) {
	return c.dialWebsocket(ctx, "/api/private/environments/"+envID+"/watch-update")
}

// BuildLog defines a build log record for a Coder environment.
type BuildLog struct {
	ID            string `db:"id" json:"id"`
	EnvironmentID string `db:"environment_id" json:"environment_id"`
	// BuildID allows the frontend to separate the logs from the old build with the logs from the new.
	BuildID string       `db:"build_id" json:"build_id"`
	Time    time.Time    `db:"time" json:"time"`
	Type    BuildLogType `db:"type" json:"type"`
	Msg     string       `db:"msg" json:"msg"`
}

// BuildLogFollowMsg wraps the base BuildLog and adds a field for collecting
// errors that may occur when follow or parsing.
type BuildLogFollowMsg struct {
	BuildLog
	Err error
}

// FollowEnvironmentBuildLog trails the build log of a Coder environment.
func (c Client) FollowEnvironmentBuildLog(ctx context.Context, envID string) (<-chan BuildLogFollowMsg, error) {
	ch := make(chan BuildLogFollowMsg)
	ws, err := c.DialEnvironmentBuildLog(ctx, envID)
	if err != nil {
		return nil, err
	}
	go func() {
		defer ws.Close(websocket.StatusNormalClosure, "normal closure")
		defer close(ch)
		for {
			var msg BuildLog
			if err := wsjson.Read(ctx, ws, &msg); err != nil {
				ch <- BuildLogFollowMsg{Err: err}
				if xerrors.Is(err, context.Canceled) || xerrors.Is(err, context.DeadlineExceeded) {
					return
				}
				continue
			}
			ch <- BuildLogFollowMsg{BuildLog: msg}
		}
	}()
	return ch, nil
}

// DialEnvironmentStats opens a websocket connection for environment stats.
func (c Client) DialEnvironmentStats(ctx context.Context, envID string) (*websocket.Conn, error) {
	return c.dialWebsocket(ctx, "/api/private/environments/"+envID+"/watch-stats")
}

// DialResourceLoad opens a websocket connection for cpu load metrics on the environment.
func (c Client) DialResourceLoad(ctx context.Context, envID string) (*websocket.Conn, error) {
	return c.dialWebsocket(ctx, "/api/private/environments/"+envID+"/watch-resource-load")
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

// WaitForEnvironmentReady will watch the build log and return when done.
func (c Client) WaitForEnvironmentReady(ctx context.Context, envID string) error {
	conn, err := c.DialEnvironmentBuildLog(ctx, envID)
	if err != nil {
		return xerrors.Errorf("%s: dial build log: %w", envID, err)
	}

	for {
		msg := buildLogMsg{}
		err := wsjson.Read(ctx, conn, &msg)
		if err != nil {
			return xerrors.Errorf("%s: reading build log msg: %w", envID, err)
		}

		if msg.Type == BuildLogTypeDone {
			return nil
		}
	}
}
