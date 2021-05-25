package coder

import (
	"context"
	"net/http"
)

// WorkspaceProviders defines all available Coder workspace provider targets.
type WorkspaceProviders struct {
	Kubernetes []KubernetesProvider `json:"kubernetes"`
}

// KubernetesProvider defines an entity capable of deploying and acting as an ingress for Coder workspaces.
type KubernetesProvider struct {
	ID                 string                  `json:"id"                  table:"-"`
	Name               string                  `json:"name"                table:"Name"`
	Status             WorkspaceProviderStatus `json:"status"              table:"Status"`
	BuiltIn            bool                    `json:"built_in"            table:"-"`
	EnvproxyAccessURL  string                  `json:"envproxy_access_url" table:"Access URL" validate:"required"`
	DevurlHost         string                  `json:"devurl_host"         table:"Devurl Host"`
	OrgWhitelist       []string                `json:"org_whitelist"       table:"-"`
	KubeProviderConfig `json:"config" table:"_"`
}

// KubeProviderConfig defines Kubernetes-specific configuration options.
type KubeProviderConfig struct {
	ClusterAddress      string `json:"cluster_address" table:"Cluster Address"`
	DefaultNamespace    string `json:"default_namespace" table:"Namespace"`
	StorageClass        string `json:"storage_class" table:"Storage Class"`
	ClusterDomainSuffix string `json:"cluster_domain_suffix" table:"Cluster Domain Suffix"`
	SSHEnabled          bool   `json:"ssh_enabled" table:"SSH Enabled"`
}

// WorkspaceProviderStatus represents the configuration state of a workspace provider.
type WorkspaceProviderStatus string

// Workspace Provider statuses.
const (
	WorkspaceProviderPending WorkspaceProviderStatus = "pending"
	WorkspaceProviderReady   WorkspaceProviderStatus = "ready"
)

// WorkspaceProviderType represents the type of workspace provider.
type WorkspaceProviderType string

// Workspace Provider types.
const (
	WorkspaceProviderTypeKubernetes WorkspaceProviderType = "kubernetes"
)

// WorkspaceProviderByID fetches a workspace provider entity by its unique ID.
func (c *DefaultClient) WorkspaceProviderByID(ctx context.Context, id string) (*KubernetesProvider, error) {
	var wp KubernetesProvider
	err := c.requestBody(ctx, http.MethodGet, "/api/private/resource-pools/"+id, nil, &wp)
	if err != nil {
		return nil, err
	}
	return &wp, nil
}

// WorkspaceProviders fetches all workspace providers known to the Coder control plane.
func (c *DefaultClient) WorkspaceProviders(ctx context.Context) (*WorkspaceProviders, error) {
	var providers WorkspaceProviders
	err := c.requestBody(ctx, http.MethodGet, "/api/private/resource-pools", nil, &providers)
	if err != nil {
		return nil, err
	}
	return &providers, nil
}
// WorkspaceProviderKubernetesCreateRequest defines the request parameters for creating a new kubernetes workspace provider.
type WorkspaceProviderKubernetesCreateRequest struct {
	Name                string   `json:"name"`
	ClusterCA           string   `json:"cluster_ca"`
	ClusterAddress      string   `json:"cluster_address"`
	SAToken             string   `json:"sa_token"`
	DefaultNamespace    string   `json:"default_namespace"`
	StorageClass        string   `json:"storage_class"`
	SSHEnabled          bool     `json:"ssh_enabled"`
	EnvironmentSA       string   `json:"environment_sa"`
	OrgAllowlist        []string `json:"org_allowlist"`
}

// WorkspaceProviderKubernetes defines a kubernetes workspace provider.
type WorkspaceProviderKubernetes struct {
	WorkspaceProviderGeneric
	Config WorkspaceProviderKubeConfig `json:"config"`
}

// WorkspaceProviderGeneric defines the generic parameters of a workspace provider.
type WorkspaceProviderGeneric struct {
	ID                string                      `json:"id"`
	Name              string                      `json:"name"`
	Status            WorkspaceProviderStatus     `json:"status"`
	BuiltIn           bool                        `json:"built_in"`
	OrgAllowlist      []string                    `json:"org_allowlist"`
	EnvproxyAccessURL string                      `json:"envproxy_access_url"`
	DevurlHost        string                      `json:"devurl_host"`
	ErrorMessage      string                      `json:"error_message"`
	CordonReason      string                      `json:"cordon_reason"`
	EnableNetV2       bool                        `json:"enable_netv2"`
}

// WorkspaceProviderKubeConfig defines the kubernetes parameters of a workspace provider.
type WorkspaceProviderKubeConfig struct {
	ClusterCA           string `json:"cluster_ca" diff:"cluster_ca"`
	ClusterAddress      string `json:"cluster_address" diff:"cluster_address"`
	SAToken             string `json:"sa_token" diff:"sa_token,private"`
	DefaultNamespace    string `json:"default_namespace" diff:"default_namespace"`
	StorageClass        string `json:"storage_class" diff:"storage_class"`
	ClusterDomainSuffix string `json:"cluster_domain_suffix" diff:"cluster_domain_suffix"`
	SSHEnabled          bool   `json:"ssh_enabled" diff:"ssh_enabled"`
	EnvironmentSA       string `json:"environment_sa" diff:"environment_sa"`
}

// CreateWorkspaceProvider creates a new WorkspaceProvider entity.
func (c *DefaultClient) CreateWorkspaceProvider(ctx context.Context, req WorkspaceProviderKubernetesCreateRequest) (*WorkspaceProviderKubernetes, error) {
	var res WorkspaceProviderKubernetes
	err := c.requestBody(ctx, http.MethodPost, "/api/private/workspace-providers", req, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// DeleteWorkspaceProviderByID deletes a workspace provider entity from the Coder control plane.
func (c *DefaultClient) DeleteWorkspaceProviderByID(ctx context.Context, id string) error {
	return c.requestBody(ctx, http.MethodDelete, "/api/private/resource-pools/"+id, nil, nil)
}

// CordoneWorkspaceProviderReq defines the request parameters for creating a new workspace provider entity.
type CordoneWorkspaceProviderReq struct {
	Reason string `json:"reason"`
}

// CordonWorkspaceProvider prevents the provider from having any more workspaces placed on it.
func (c *DefaultClient) CordonWorkspaceProvider(ctx context.Context, id, reason string) error {
	req := CordoneWorkspaceProviderReq{Reason: reason}
	err := c.requestBody(ctx, http.MethodPost, "/api/private/resource-pools/"+id+"/cordon", req, nil)
	if err != nil {
		return err
	}
	return nil
}

// UnCordonWorkspaceProvider changes an existing cordoned providers status to 'Ready';
// allowing it to continue creating new workspaces and provisioning resources for them.
func (c *DefaultClient) UnCordonWorkspaceProvider(ctx context.Context, id string) error {
	err := c.requestBody(ctx, http.MethodPost, "/api/private/resource-pools/"+id+"/uncordon", nil, nil)
	if err != nil {
		return err
	}
	return nil
}

// RenameWorkspaceProviderReq defines the request parameters for changing a workspace provider name.
type RenameWorkspaceProviderReq struct {
	Name string `json:"name"`
}

// RenameWorkspaceProvider changes an existing cordoned providers name field.
func (c *DefaultClient) RenameWorkspaceProvider(ctx context.Context, id string, name string) error {
	req := RenameWorkspaceProviderReq{Name: name}
	err := c.requestBody(ctx, http.MethodPatch, "/api/private/resource-pools/"+id, req, nil)
	if err != nil {
		return err
	}
	return nil
}
