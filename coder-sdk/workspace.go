package coder

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"cdr.dev/wsep"
	"golang.org/x/xerrors"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

// Workspace describes a Coder workspace.
type Workspace struct {
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
	GPUs             int              `json:"gpus"               table:"-"`
	Updating         bool             `json:"updating"           table:"-"`
	LatestStat       WorkspaceStat    `json:"latest_stat"        table:"Status"`
	RebuildMessages  []RebuildMessage `json:"rebuild_messages"   table:"-"`
	CreatedAt        time.Time        `json:"created_at"         table:"-"`
	UpdatedAt        time.Time        `json:"updated_at"         table:"-"`
	LastOpenedAt     time.Time        `json:"last_opened_at"     table:"-"`
	LastConnectionAt time.Time        `json:"last_connection_at" table:"-"`
	AutoOffThreshold Duration         `json:"auto_off_threshold" table:"-"`
	UseContainerVM   bool             `json:"use_container_vm"   table:"CVM"`
	ResourcePoolID   string           `json:"resource_pool_id"   table:"-"`
}

// RebuildMessage defines the message shown when a Workspace requires a rebuild for it can be accessed.
type RebuildMessage struct {
	Text             string   `json:"text"`
	Required         bool     `json:"required"`
	AutoOffThreshold Duration `json:"auto_off_threshold"`
}

// WorkspaceStat represents the state of a workspace.
type WorkspaceStat struct {
	Time            time.Time       `json:"time"`
	LastOnline      time.Time       `json:"last_online"`
	ContainerStatus WorkspaceStatus `json:"container_status"`
	StatError       string          `json:"stat_error"`
	CPUUsage        float32         `json:"cpu_usage"`
	MemoryTotal     int64           `json:"memory_total"`
	MemoryUsage     float32         `json:"memory_usage"`
	DiskTotal       int64           `json:"disk_total"`
	DiskUsed        int64           `json:"disk_used"`
}

func (e WorkspaceStat) String() string { return string(e.ContainerStatus) }

// WorkspaceStatus refers to the states of a workspace.
type WorkspaceStatus string

// The following represent the possible workspace container states.
const (
	WorkspaceCreating WorkspaceStatus = "CREATING"
	WorkspaceOff      WorkspaceStatus = "OFF"
	WorkspaceOn       WorkspaceStatus = "ON"
	WorkspaceFailed   WorkspaceStatus = "FAILED"
	WorkspaceUnknown  WorkspaceStatus = "UNKNOWN"
)

// CreateWorkspaceRequest is used to configure a new workspace.
type CreateWorkspaceRequest struct {
	Name            string  `json:"name"`
	ImageID         string  `json:"image_id"`
	OrgID           string  `json:"org_id"`
	ImageTag        string  `json:"image_tag"`
	CPUCores        float32 `json:"cpu_cores"`
	MemoryGB        float32 `json:"memory_gb"`
	DiskGB          int     `json:"disk_gb"`
	GPUs            int     `json:"gpus"`
	UseContainerVM  bool    `json:"use_container_vm"`
	ResourcePoolID  string  `json:"resource_pool_id"`
	Namespace       string  `json:"namespace"`
	EnableAutoStart bool    `json:"autostart_enabled"`

	// TemplateID comes from the parse template route on cemanager.
	TemplateID string `json:"template_id,omitempty"`
}

// CreateWorkspace sends a request to create a workspace.
func (c *DefaultClient) CreateWorkspace(ctx context.Context, req CreateWorkspaceRequest) (*Workspace, error) {
	var workspace Workspace
	if err := c.requestBody(ctx, http.MethodPost, "/api/v0/workspaces", req, &workspace); err != nil {
		return nil, err
	}
	return &workspace, nil
}

// ParseTemplateRequest parses a template. If Local is a non-nil reader
// it will obviate any other fields on the request.
type ParseTemplateRequest struct {
	RepoURL  string    `json:"repo_url"`
	Ref      string    `json:"ref"`
	Filepath string    `json:"filepath"`
	OrgID    string    `json:"-"`
	Local    io.Reader `json:"-"`
}

// TemplateVersion is a workspace template.
// For now, let's not interpret it on the CLI level. We just need
// to forward this as part of the create workspace request.
type TemplateVersion struct {
	ID         string `json:"id"`
	TemplateID string `json:"template_id"`
	// FileHash is the sha256 hash of the template's file contents.
	FileHash string `json:"file_hash"`
	// Commit is the git commit from which the template was derived.
	Commit        string    `json:"commit"`
	CommitMessage string    `json:"commit_message"`
	CreatedAt     time.Time `json:"created_at"`
}

// ParseTemplate parses a template config. It support both remote repositories and local files.
// If a local file is specified then all other values in the request are ignored.
func (c *DefaultClient) ParseTemplate(ctx context.Context, req ParseTemplateRequest) (*TemplateVersion, error) {
	const path = "/api/private/workspaces/template/parse"
	var (
		tpl     TemplateVersion
		opts    []requestOption
		headers = http.Header{}
		query   = url.Values{}
	)

	query.Set("org-id", req.OrgID)

	opts = append(opts, withQueryParams(query))

	if req.Local == nil {
		if err := c.requestBody(ctx, http.MethodPost, path, req, &tpl, opts...); err != nil {
			return &tpl, err
		}
		return &tpl, nil
	}

	headers.Set("Content-Type", "application/octet-stream")
	opts = append(opts, withBody(req.Local), withHeaders(headers))

	err := c.requestBody(ctx, http.MethodPost, path, nil, &tpl, opts...)
	if err != nil {
		return &tpl, err
	}

	return &tpl, nil
}

// CreateWorkspaceFromRepo sends a request to create a workspace from a repository.
func (c *DefaultClient) CreateWorkspaceFromRepo(ctx context.Context, orgID string, req TemplateVersion) (*Workspace, error) {
	var workspace Workspace
	if err := c.requestBody(ctx, http.MethodPost, "/api/private/orgs/"+orgID+"/workspaces/from-repo", req, &workspace); err != nil {
		return nil, err
	}
	return &workspace, nil
}

// Workspaces lists workspaces returned by the given filter.
// TODO: add the filter options, explore performance issue.
func (c *DefaultClient) Workspaces(ctx context.Context) ([]Workspace, error) {
	var workspaces []Workspace
	if err := c.requestBody(ctx, http.MethodGet, "/api/v0/workspaces", nil, &workspaces); err != nil {
		return nil, err
	}
	return workspaces, nil
}

// UserWorkspacesByOrganization gets the list of workspaces owned by the given user.
func (c *DefaultClient) UserWorkspacesByOrganization(ctx context.Context, userID, orgID string) ([]Workspace, error) {
	var (
		workspaces []Workspace
		query      = url.Values{}
	)

	query.Add("orgs", orgID)
	query.Add("users", userID)

	if err := c.requestBody(ctx, http.MethodGet, "/api/v0/workspaces", nil, &workspaces, withQueryParams(query)); err != nil {
		return nil, err
	}
	return workspaces, nil
}

// DeleteWorkspace deletes the workspace.
func (c *DefaultClient) DeleteWorkspace(ctx context.Context, workspaceID string) error {
	return c.requestBody(ctx, http.MethodDelete, "/api/v0/workspaces/"+workspaceID, nil, nil)
}

// StopWorkspace stops the workspace.
func (c *DefaultClient) StopWorkspace(ctx context.Context, workspaceID string) error {
	return c.requestBody(ctx, http.MethodPut, "/api/v0/workspaces/"+workspaceID+"/stop", nil, nil)
}

// UpdateWorkspaceReq defines the update operation, only setting
// nil-fields.
type UpdateWorkspaceReq struct {
	ImageID    *string  `json:"image_id"`
	ImageTag   *string  `json:"image_tag"`
	CPUCores   *float32 `json:"cpu_cores"`
	MemoryGB   *float32 `json:"memory_gb"`
	DiskGB     *int     `json:"disk_gb"`
	GPUs       *int     `json:"gpus"`
	TemplateID *string  `json:"template_id"`
}

// RebuildWorkspace requests that the given workspaceID is rebuilt with no changes to its specification.
func (c *DefaultClient) RebuildWorkspace(ctx context.Context, workspaceID string) error {
	return c.requestBody(ctx, http.MethodPatch, "/api/v0/workspaces/"+workspaceID, UpdateWorkspaceReq{}, nil)
}

// EditWorkspace modifies the workspace specification and initiates a rebuild.
func (c *DefaultClient) EditWorkspace(ctx context.Context, workspaceID string, req UpdateWorkspaceReq) error {
	return c.requestBody(ctx, http.MethodPatch, "/api/v0/workspaces/"+workspaceID, req, nil)
}

// DialWsep dials a workspace's command execution interface
// See https://github.com/cdr/wsep for details.
func (c *DefaultClient) DialWsep(ctx context.Context, baseURL *url.URL, workspaceID string) (*websocket.Conn, error) {
	return c.dialWebsocket(ctx, "/proxy/workspaces/"+workspaceID+"/wsep", withBaseURL(baseURL))
}

// DialExecutor gives a remote execution interface for performing commands
// inside a workspace.
func (c *DefaultClient) DialExecutor(ctx context.Context, baseURL *url.URL, workspaceID string) (wsep.Execer, error) {
	ws, err := c.DialWsep(ctx, baseURL, workspaceID)
	if err != nil {
		return nil, err
	}
	return wsep.RemoteExecer(ws), nil
}

// DialIDEStatus opens a websocket connection for cpu load metrics on the workspace.
func (c *DefaultClient) DialIDEStatus(ctx context.Context, baseURL *url.URL, workspaceID string) (*websocket.Conn, error) {
	return c.dialWebsocket(ctx, "/proxy/workspaces/"+workspaceID+"/ide/api/status", withBaseURL(baseURL))
}

// DialWorkspaceBuildLog opens a websocket connection for the workspace build log messages.
func (c *DefaultClient) DialWorkspaceBuildLog(ctx context.Context, workspaceID string) (*websocket.Conn, error) {
	return c.dialWebsocket(ctx, "/api/private/workspaces/"+workspaceID+"/watch-update")
}

// BuildLog defines a build log record for a Coder workspace.
type BuildLog struct {
	ID          string `db:"id" json:"id"`
	WorkspaceID string `db:"workspace_id" json:"workspace_id"`
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

// FollowWorkspaceBuildLog trails the build log of a Coder workspace.
func (c *DefaultClient) FollowWorkspaceBuildLog(ctx context.Context, workspaceID string) (<-chan BuildLogFollowMsg, error) {
	ch := make(chan BuildLogFollowMsg)
	ws, err := c.DialWorkspaceBuildLog(ctx, workspaceID)
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

// DialWorkspaceStats opens a websocket connection for workspace stats.
func (c *DefaultClient) DialWorkspaceStats(ctx context.Context, workspaceID string) (*websocket.Conn, error) {
	return c.dialWebsocket(ctx, "/api/private/workspaces/"+workspaceID+"/watch-stats")
}

// DialResourceLoad opens a websocket connection for cpu load metrics on the workspace.
func (c *DefaultClient) DialResourceLoad(ctx context.Context, workspaceID string) (*websocket.Conn, error) {
	return c.dialWebsocket(ctx, "/api/private/workspaces/"+workspaceID+"/watch-resource-load")
}

// BuildLogType describes the type of an event.
type BuildLogType string

const (
	// BuildLogTypeStart signals that a new build log has begun.
	BuildLogTypeStart BuildLogType = "start"
	// BuildLogTypeStage is a stage-level event for a workspace.
	// It can be thought of as a major step in the workspace's
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

// WaitForWorkspaceReady will watch the build log and return when done.
func (c *DefaultClient) WaitForWorkspaceReady(ctx context.Context, workspaceID string) error {
	conn, err := c.DialWorkspaceBuildLog(ctx, workspaceID)
	if err != nil {
		return xerrors.Errorf("%s: dial build log: %w", workspaceID, err)
	}

	for {
		msg := buildLogMsg{}
		err := wsjson.Read(ctx, conn, &msg)
		if err != nil {
			return xerrors.Errorf("%s: reading build log msg: %w", workspaceID, err)
		}

		if msg.Type == BuildLogTypeDone {
			return nil
		}
	}
}

// WorkspaceByID get the details of a workspace by its id.
func (c *DefaultClient) WorkspaceByID(ctx context.Context, id string) (*Workspace, error) {
	var workspace Workspace
	if err := c.requestBody(ctx, http.MethodGet, "/api/v0/workspaces/"+id, nil, &workspace); err != nil {
		return nil, err
	}
	return &workspace, nil
}

// WorkspacesByWorkspaceProvider returns all workspaces that belong to a particular workspace provider.
func (c *DefaultClient) WorkspacesByWorkspaceProvider(ctx context.Context, wpID string) ([]Workspace, error) {
	var workspaces []Workspace
	if err := c.requestBody(ctx, http.MethodGet, "/api/private/resource-pools/"+wpID+"/workspaces", nil, &workspaces); err != nil {
		return nil, err
	}
	return workspaces, nil
}

const (
	// SkipTemplateOrg allows skipping checks on organizations.
	SkipTemplateOrg = "SKIP_ORG"
)

type TemplateScope string

const (
	// TemplateScopeSite is the scope for a site wide policy template.
	TemplateScopeSite = "site"
)

type SetPolicyTemplateRequest struct {
	TemplateID string `json:"template_id"`
	Type       string `json:"type"` // site, org
}

type SetPolicyTemplateResponse struct {
	MergeConflicts []*WorkspaceTemplateMergeConflict `json:"merge_conflicts"`
}

type WorkspaceTemplateMergeConflict struct {
	WorkspaceID             string    `json:"workspace_id"`
	CurrentTemplateWarnings []string  `json:"current_template_warnings"`
	CurrentTemplateError    *TplError `json:"current_template_errors"`
	LatestTemplateWarnings  []string  `json:"latest_template_warnings"`
	LatestTemplateError     *TplError `json:"latest_template_errors"`
	CurrentTemplateIsLatest bool      `json:"current_template_is_latest"`
	Message                 string    `json:"message"`
}

func (mc WorkspaceTemplateMergeConflict) String() string {
	var sb strings.Builder

	if mc.Message != "" {
		sb.WriteString(mc.Message)
	}

	currentConflicts := len(mc.CurrentTemplateWarnings) != 0 || mc.CurrentTemplateError != nil
	updateConflicts := len(mc.LatestTemplateWarnings) != 0 || mc.LatestTemplateError != nil

	if !currentConflicts && !updateConflicts {
		sb.WriteString("No workspace conflicts\n")
		return sb.String()
	}

	if currentConflicts {
		if len(mc.CurrentTemplateWarnings) != 0 {
			fmt.Fprintf(&sb, "Warnings: \n%s\n", strings.Join(mc.CurrentTemplateWarnings, "\n"))
		}
		if mc.CurrentTemplateError != nil {
			fmt.Fprintf(&sb, "Errors: \n%s\n", strings.Join(mc.CurrentTemplateError.Msgs, "\n"))
		}
	}

	if !mc.CurrentTemplateIsLatest && updateConflicts {
		sb.WriteString("If workspace is updated to the latest template:\n")
		if len(mc.LatestTemplateWarnings) != 0 {
			fmt.Fprintf(&sb, "Warnings: \n%s\n", strings.Join(mc.LatestTemplateWarnings, "\n"))
		}
		if mc.LatestTemplateError != nil {
			fmt.Fprintf(&sb, "Errors: \n%s\n", strings.Join(mc.LatestTemplateError.Msgs, "\n"))
		}
	}

	return sb.String()
}

type WorkspaceTemplateMergeConflicts []*WorkspaceTemplateMergeConflict

func (mcs WorkspaceTemplateMergeConflicts) Summary() string {
	var (
		sb              strings.Builder
		currentWarnings int
		updateWarnings  int
		currentErrors   int
		updateErrors    int
	)

	for _, mc := range mcs {
		if len(mc.CurrentTemplateWarnings) != 0 {
			currentWarnings++
		}
		if len(mc.LatestTemplateWarnings) != 0 {
			updateWarnings++
		}
		if mc.CurrentTemplateError != nil {
			currentErrors++
		}
		if mc.LatestTemplateError != nil {
			updateErrors++
		}
	}

	if currentErrors == 0 && updateErrors == 0 && currentWarnings == 0 && updateWarnings == 0 {
		sb.WriteString("No workspace conflicts\n")
		return sb.String()
	}

	if currentErrors != 0 {
		fmt.Fprintf(&sb, "%d workspaces will not be able to be rebuilt\n", currentErrors)
	}
	if updateErrors != 0 {
		fmt.Fprintf(&sb, "%d workspaces will not be able to be rebuilt if updated to the latest version\n", updateErrors)
	}
	if currentWarnings != 0 {
		fmt.Fprintf(&sb, "%d workspaces will be impacted\n", currentWarnings)
	}
	if updateWarnings != 0 {
		fmt.Fprintf(&sb, "%d workspaces will be impacted if updated to the latest version\n", updateWarnings)
	}

	return sb.String()
}

type TplError struct {
	// Msgs are the human facing strings to present to the user. Since there can be multiple
	// problems with a template, there might be multiple strings
	Msgs []string `json:"messages"`
}

func (c *DefaultClient) SetPolicyTemplate(ctx context.Context, templateID string, templateScope TemplateScope, dryRun bool) (*SetPolicyTemplateResponse, error) {
	var (
		resp  SetPolicyTemplateResponse
		query = url.Values{}
	)

	req := SetPolicyTemplateRequest{
		TemplateID: templateID,
		Type:       string(templateScope),
	}

	if dryRun {
		query.Set("dry-run", "true")
	}

	if err := c.requestBody(ctx, http.MethodPost, "/api/private/workspaces/template/policy", req, &resp, withQueryParams(query)); err != nil {
		return nil, err
	}

	return &resp, nil
}
