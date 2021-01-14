package coder

import (
	"context"
	"net/http"
)

// ResourcePool defines an entity capable of deploying and acting as an ingress for Coder environments.
type ResourcePool struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	Local               bool     `json:"local"`
	ClusterAddress      string   `json:"cluster_address"`
	DefaultNamespace    string   `json:"default_namespace"`
	StorageClass        string   `json:"storage_class"`
	ClusterDomainSuffix string   `json:"cluster_domain_suffix"`
	DevurlHost          string   `json:"devurl_host"`
	NamespaceWhitelist  []string `json:"namespace_whitelist"`
	OrgWhitelist        []string `json:"org_whitelist"`
}

// ResourcePoolByID fetches a resource pool entity by its unique ID.
func (c *Client) ResourcePoolByID(ctx context.Context, id string) (*ResourcePool, error) {
	var rp ResourcePool
	if err := c.requestBody(ctx, http.MethodGet, "/api/private/resource-pools/"+id, nil, &rp); err != nil {
		return nil, err
	}
	return &rp, nil
}

// DeleteResourcePoolByID deletes a resource pool entity from the Coder control plane.
func (c *Client) DeleteResourcePoolByID(ctx context.Context, id string) error {
	return c.requestBody(ctx, http.MethodDelete, "/api/private/resource-pools/"+id, nil, nil)
}

// ResourcePools fetches all resource pools known to the Coder control plane.
func (c *Client) ResourcePools(ctx context.Context) ([]ResourcePool, error) {
	var pools []ResourcePool
	if err := c.requestBody(ctx, http.MethodGet, "/api/private/resource-pools", nil, &pools); err != nil {
		return nil, err
	}
	return pools, nil
}
