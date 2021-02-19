package coder

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

// Registry defines an image registry configuration.
type Registry struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	FriendlyName   string    `json:"friendly_name"`
	Registry       string    `json:"registry"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Registries fetches all registries in an organization.
func (c *defaultClient) Registries(ctx context.Context, orgID string) ([]Registry, error) {
	var (
		r     []Registry
		query = url.Values{}
	)

	query.Set("org", orgID)

	if err := c.requestBody(ctx, http.MethodGet, "/api/v0/registries", nil, &r, withQueryParams(query)); err != nil {
		return nil, err
	}
	return r, nil
}

// RegistryByID fetches a registry resource by its ID.
func (c *defaultClient) RegistryByID(ctx context.Context, registryID string) (*Registry, error) {
	var r Registry
	if err := c.requestBody(ctx, http.MethodGet, "/api/v0/registries/"+registryID, nil, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// UpdateRegistryReq defines the requests parameters for a partial update of a registry resource.
type UpdateRegistryReq struct {
	Registry     *string `json:"registry"`
	FriendlyName *string `json:"friendly_name"`
	Username     *string `json:"username"`
	Password     *string `json:"password"`
}

// UpdateRegistry applies a partial update to a registry resource.
func (c *defaultClient) UpdateRegistry(ctx context.Context, registryID string, req UpdateRegistryReq) error {
	return c.requestBody(ctx, http.MethodPatch, "/api/v0/registries/"+registryID, req, nil)
}

// DeleteRegistry deletes a registry resource by its ID.
func (c *defaultClient) DeleteRegistry(ctx context.Context, registryID string) error {
	return c.requestBody(ctx, http.MethodDelete, "/api/v0/registries/"+registryID, nil, nil)
}
