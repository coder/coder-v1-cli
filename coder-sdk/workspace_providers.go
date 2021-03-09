package coder

import (
	"context"
	"net/http"
)

// WorkspaceProviders defines all available Coder workspace provider targets.
type WorkspaceProviders struct {
	Kubernetes []KubernetesProvider `json:"kubernetes"`
}

// KubernetesProvider defines an entity capable of deploying and acting as an ingress for Coder environments.
type KubernetesProvider struct {
	ID                 string                  `json:"id" table:"-"`
	Name               string                  `json:"name" table:"Name"`
	Status             WorkspaceProviderStatus `json:"status" table:"Status"`
	Local              bool                    `json:"local" table:"-"`
	EnvproxyAccessURL  string                  `json:"envproxy_access_url" validate:"required" table:"Access URL"`
	DevurlHost         string                  `json:"devurl_host" table:"Devurl Host"`
	OrgWhitelist       []string                `json:"org_whitelist" table:"-"`
	KubeProviderConfig `json:"config"`
}

// KubeProviderConfig defines Kubernetes-specific configuration options.
type KubeProviderConfig struct {
	ClusterAddress      string   `json:"cluster_address" table:"Cluster Address"`
	DefaultNamespace    string   `json:"default_namespace" table:"Namespace"`
	StorageClass        string   `json:"storage_class" table:"Storage Class"`
	ClusterDomainSuffix string   `json:"cluster_domain_suffix" table:"Cluster Domain Suffix"`
	SSHEnabled          bool     `json:"ssh_enabled" table:"SSH Enabled"`
	NamespaceWhitelist  []string `json:"namespace_whitelist" table:"Namespace Allowlist"`
}

// WorkspaceProviderStatus represents the configuration state of a workspace provider.
type WorkspaceProviderStatus string

// Workspace Provider statuses.
const (
	WorkspaceProviderPending WorkspaceProviderStatus = "pending"
	WorkspaceProviderReady   WorkspaceProviderStatus = "ready"
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

// CreateWorkspaceProviderReq defines the request parameters for creating a new workspace provider entity.
type CreateWorkspaceProviderReq struct {
	Name string `json:"name"`
}

// CreateWorkspaceProviderRes defines the response from creating a new workspace provider entity.
type CreateWorkspaceProviderRes struct {
	ID            string                  `json:"id" table:"ID"`
	Name          string                  `json:"name" table:"Name"`
	Status        WorkspaceProviderStatus `json:"status" table:"Status"`
	EnvproxyToken string                  `json:"envproxy_token" table:"Envproxy Token"`
}

// CreateWorkspaceProvider creates a new WorkspaceProvider entity.
func (c *DefaultClient) CreateWorkspaceProvider(ctx context.Context, req CreateWorkspaceProviderReq) (*CreateWorkspaceProviderRes, error) {
	var res CreateWorkspaceProviderRes
	err := c.requestBody(ctx, http.MethodPost, "/api/private/resource-pools", req, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// DeleteWorkspaceProviderByID deletes a workspace provider entity from the Coder control plane.
func (c *DefaultClient) DeleteWorkspaceProviderByID(ctx context.Context, id string) error {
	return c.requestBody(ctx, http.MethodDelete, "/api/private/resource-pools/"+id, nil, nil)
}
