package coder

import (
	"context"
	"net/url"

	"cdr.dev/wsep"
	"nhooyr.io/websocket"
)

// Client wraps the Coder HTTP API.
// This is an interface to allow for mocking of coder-sdk client usage.
type Client interface {
	// PushActivity pushes CLI activity to Coder.
	PushActivity(ctx context.Context, source, workspaceID string) error

	// Me gets the details of the authenticated user.
	Me(ctx context.Context) (*User, error)

	// UserByID get the details of a user by their id.
	UserByID(ctx context.Context, id string) (*User, error)

	// SSHKey gets the current SSH kepair of the authenticated user.
	SSHKey(ctx context.Context) (*SSHKey, error)

	// Users gets the list of user accounts.
	Users(ctx context.Context) ([]User, error)

	// UserByEmail gets a user by email.
	UserByEmail(ctx context.Context, email string) (*User, error)

	// UpdateUser applyes the partial update to the given user.
	UpdateUser(ctx context.Context, userID string, req UpdateUserReq) error

	// UpdateUXState applies a partial update of the user's UX State.
	UpdateUXState(ctx context.Context, userID string, uxsPartial map[string]interface{}) error

	// CreateUser creates a new user account.
	CreateUser(ctx context.Context, req CreateUserReq) error

	// DeleteUser deletes a user account.
	DeleteUser(ctx context.Context, userID string) error

	// SiteConfigAuth fetches the sitewide authentication configuration.
	SiteConfigAuth(ctx context.Context) (*ConfigAuth, error)

	// PutSiteConfigAuth sets the sitewide authentication configuration.
	PutSiteConfigAuth(ctx context.Context, req ConfigAuth) error

	// SiteConfigOAuth fetches the sitewide git provider OAuth configuration.
	SiteConfigOAuth(ctx context.Context) (*ConfigOAuth, error)

	// PutSiteConfigOAuth sets the sitewide git provider OAuth configuration.
	PutSiteConfigOAuth(ctx context.Context, req ConfigOAuth) error

	// SiteSetupModeEnabled fetches the current setup_mode state of a Coder deployment.
	SiteSetupModeEnabled(ctx context.Context) (bool, error)

	// SiteConfigExtensionMarketplace fetches the extension marketplace configuration.
	SiteConfigExtensionMarketplace(ctx context.Context) (*ConfigExtensionMarketplace, error)

	// PutSiteConfigExtensionMarketplace sets the extension marketplace configuration.
	PutSiteConfigExtensionMarketplace(ctx context.Context, req ConfigExtensionMarketplace) error

	// SiteConfigWorkspaces fetches the workspace configuration.
	SiteConfigWorkspaces(ctx context.Context) (*ConfigWorkspaces, error)

	// DeleteDevURL deletes the specified devurl.
	DeleteDevURL(ctx context.Context, workspaceID, urlID string) error

	// CreateDevURL inserts a new devurl for the authenticated user.
	CreateDevURL(ctx context.Context, workspaceID string, req CreateDevURLReq) error

	// DevURLs fetches the Dev URLs for a given workspace.
	DevURLs(ctx context.Context, workspaceID string) ([]DevURL, error)

	// PutDevURL updates an existing devurl for the authenticated user.
	PutDevURL(ctx context.Context, workspaceID, urlID string, req PutDevURLReq) error

	// CreateWorkspace sends a request to create a workspace.
	CreateWorkspace(ctx context.Context, req CreateWorkspaceRequest) (*Workspace, error)

	// ParseTemplate parses a template config. It support both remote repositories and local files.
	// If a local file is specified then all other values in the request are ignored.
	ParseTemplate(ctx context.Context, req ParseTemplateRequest) (*TemplateVersion, error)

	// CreateWorkspaceFromRepo sends a request to create a workspace from a repository.
	CreateWorkspaceFromRepo(ctx context.Context, orgID string, req TemplateVersion) (*Workspace, error)

	// Workspaces lists workspaces returned by the given filter.
	Workspaces(ctx context.Context) ([]Workspace, error)

	// UserWorkspacesByOrganization gets the list of workspaces owned by the given user.
	UserWorkspacesByOrganization(ctx context.Context, userID, orgID string) ([]Workspace, error)

	// DeleteWorkspace deletes the workspace.
	DeleteWorkspace(ctx context.Context, workspaceID string) error

	// StopWorkspace stops the workspace.
	StopWorkspace(ctx context.Context, workspaceID string) error

	// RebuildWorkspace requests that the given workspaceID is rebuilt with no changes to its specification.
	RebuildWorkspace(ctx context.Context, workspaceID string) error

	// EditWorkspace modifies the workspace specification and initiates a rebuild.
	EditWorkspace(ctx context.Context, workspaceID string, req UpdateWorkspaceReq) error

	// DialWsep dials a workspace's command execution interface
	// See https://github.com/cdr/wsep for details.
	DialWsep(ctx context.Context, baseURL *url.URL, workspaceID string) (*websocket.Conn, error)

	// DialExecutor gives a remote execution interface for performing commands inside a workspace.
	DialExecutor(ctx context.Context, baseURL *url.URL, workspaceID string) (wsep.Execer, error)

	// DialIDEStatus opens a websocket connection for cpu load metrics on the workspace.
	DialIDEStatus(ctx context.Context, baseURL *url.URL, workspaceID string) (*websocket.Conn, error)

	// DialWorkspaceBuildLog opens a websocket connection for the workspace build log messages.
	DialWorkspaceBuildLog(ctx context.Context, workspaceID string) (*websocket.Conn, error)

	// FollowWorkspaceBuildLog trails the build log of a Coder workspace.
	FollowWorkspaceBuildLog(ctx context.Context, workspaceID string) (<-chan BuildLogFollowMsg, error)

	// DialWorkspaceStats opens a websocket connection for workspace stats.
	DialWorkspaceStats(ctx context.Context, workspaceID string) (*websocket.Conn, error)

	// DialResourceLoad opens a websocket connection for cpu load metrics on the workspace.
	DialResourceLoad(ctx context.Context, workspaceID string) (*websocket.Conn, error)

	// WaitForWorkspaceReady will watch the build log and return when done.
	WaitForWorkspaceReady(ctx context.Context, workspaceID string) error

	// WorkspaceByID get the details of a workspace by its id.
	WorkspaceByID(ctx context.Context, id string) (*Workspace, error)

	// WorkspacesByWorkspaceProvider returns workspaces that belong to a particular workspace provider.
	WorkspacesByWorkspaceProvider(ctx context.Context, wpID string) ([]Workspace, error)

	// ImportImage creates a new image and optionally a new registry.
	ImportImage(ctx context.Context, req ImportImageReq) (*Image, error)

	// ImageByID returns an image entity, fetched by its ID.
	ImageByID(ctx context.Context, id string) (*Image, error)

	// OrganizationImages returns all of the images imported for orgID.
	OrganizationImages(ctx context.Context, orgID string) ([]Image, error)

	// UpdateImage applies a partial update to an image resource.
	UpdateImage(ctx context.Context, imageID string, req UpdateImageReq) error

	// UpdateImageTags refreshes the latest digests for all tags of the image.
	UpdateImageTags(ctx context.Context, imageID string) error

	// Organizations gets all Organizations.
	Organizations(ctx context.Context) ([]Organization, error)

	// OrganizationByID get the Organization by its ID.
	OrganizationByID(ctx context.Context, orgID string) (*Organization, error)

	// OrganizationMembers get all members of the given organization.
	OrganizationMembers(ctx context.Context, orgID string) ([]OrganizationUser, error)

	// UpdateOrganization applys a partial update of an Organization resource.
	UpdateOrganization(ctx context.Context, orgID string, req UpdateOrganizationReq) error

	// CreateOrganization creates a new Organization in Coder.
	CreateOrganization(ctx context.Context, req CreateOrganizationReq) error

	// DeleteOrganization deletes an organization.
	DeleteOrganization(ctx context.Context, orgID string) error

	// Registries fetches all registries in an organization.
	Registries(ctx context.Context, orgID string) ([]Registry, error)

	// RegistryByID fetches a registry resource by its ID.
	RegistryByID(ctx context.Context, registryID string) (*Registry, error)

	// UpdateRegistry applies a partial update to a registry resource.
	UpdateRegistry(ctx context.Context, registryID string, req UpdateRegistryReq) error

	// DeleteRegistry deletes a registry resource by its ID.
	DeleteRegistry(ctx context.Context, registryID string) error

	// CreateImageTag creates a new image tag resource.
	CreateImageTag(ctx context.Context, imageID string, req CreateImageTagReq) (*ImageTag, error)

	// DeleteImageTag deletes an image tag resource.
	DeleteImageTag(ctx context.Context, imageID, tag string) error

	// ImageTags fetch all image tags.
	ImageTags(ctx context.Context, imageID string) ([]ImageTag, error)

	// ImageTagByID fetch an image tag by ID.
	ImageTagByID(ctx context.Context, imageID, tagID string) (*ImageTag, error)

	// CreateAPIToken creates a new APIToken for making authenticated requests to Coder.
	CreateAPIToken(ctx context.Context, userID string, req CreateAPITokenReq) (string, error)

	// APITokens fetches all APITokens owned by the given user.
	APITokens(ctx context.Context, userID string) ([]APIToken, error)

	// APITokenByID fetches the metadata for a given APIToken.
	APITokenByID(ctx context.Context, userID, tokenID string) (*APIToken, error)

	// DeleteAPIToken deletes an APIToken.
	DeleteAPIToken(ctx context.Context, userID, tokenID string) error

	// RegenerateAPIToken regenerates the given APIToken and returns the new value.
	RegenerateAPIToken(ctx context.Context, userID, tokenID string) (string, error)

	// APIVersion parses the coder-version http header from an authenticated request.
	APIVersion(ctx context.Context) (string, error)

	// WorkspaceProviderByID fetches a workspace provider entity by its unique ID.
	WorkspaceProviderByID(ctx context.Context, id string) (*KubernetesProvider, error)

	// WorkspaceProviders fetches all workspace providers known to the Coder control plane.
	WorkspaceProviders(ctx context.Context) (*WorkspaceProviders, error)

	// CreateWorkspaceProvider creates a new WorkspaceProvider entity.
	CreateWorkspaceProvider(ctx context.Context, req CreateWorkspaceProviderReq) (*CreateWorkspaceProviderRes, error)

	// DeleteWorkspaceProviderByID deletes a workspace provider entity from the Coder control plane.
	DeleteWorkspaceProviderByID(ctx context.Context, id string) error

	// Token returns the API Token used to authenticate.
	Token() string

	// BaseURL returns the BaseURL configured for this Client.
	BaseURL() url.URL

	// CordonWorkspaceProvider prevents the provider from having any more workspaces placed on it.
	CordonWorkspaceProvider(ctx context.Context, id, reason string) error

	// UnCordonWorkspaceProvider changes an existing cordoned providers status to 'Ready';
	// allowing it to continue creating new workspaces and provisioning resources for them.
	UnCordonWorkspaceProvider(ctx context.Context, id string) error

	// RenameWorkspaceProvider changes an existing providers name field.
	RenameWorkspaceProvider(ctx context.Context, id string, name string) error
}
