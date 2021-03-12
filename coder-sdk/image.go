package coder

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

// Image describes a Coder Image.
type Image struct {
	ID              string    `json:"id"                    table:"-"`
	OrganizationID  string    `json:"organization_id"       table:"-"`
	Repository      string    `json:"repository"            table:"Repository"`
	Description     string    `json:"description"           table:"-"`
	URL             string    `json:"url"                   table:"-"` // User-supplied URL for image.
	Registry        *Registry `json:"registry"              table:"-"`
	DefaultTag      *ImageTag `json:"default_tag"           table:"DefaultTag"`
	DefaultCPUCores float32   `json:"default_cpu_cores"     table:"DefaultCPUCores"`
	DefaultMemoryGB float32   `json:"default_memory_gb"     table:"DefaultMemoryGB"`
	DefaultDiskGB   int       `json:"default_disk_gb"       table:"DefaultDiskGB"`
	Deprecated      bool      `json:"deprecated"            table:"-"`
	CreatedAt       time.Time `json:"created_at"            table:"-"`
	UpdatedAt       time.Time `json:"updated_at"            table:"-"`
}

// NewRegistryRequest describes a docker registry used in importing an image.
type NewRegistryRequest struct {
	FriendlyName string `json:"friendly_name"`
	Registry     string `json:"registry"`
	Username     string `json:"username"`
	Password     string `json:"password"`
}

// ImportImageReq is used to import new images and registries into Coder.
type ImportImageReq struct {
	RegistryID      *string             `json:"registry_id"`  // Used to import images to existing registries.
	NewRegistry     *NewRegistryRequest `json:"new_registry"` // Used when adding a new registry.
	Repository      string              `json:"repository"`   // Refers to the image. Ex: "codercom/ubuntu".
	OrgID           string              `json:"org_id"`
	Tag             string              `json:"tag"`
	DefaultCPUCores float32             `json:"default_cpu_cores"`
	DefaultMemoryGB int                 `json:"default_memory_gb"`
	DefaultDiskGB   int                 `json:"default_disk_gb"`
	Description     string              `json:"description"`
	URL             string              `json:"url"`
}

// UpdateImageReq defines the requests parameters for a partial update of an image resource.
type UpdateImageReq struct {
	DefaultCPUCores *float32 `json:"default_cpu_cores"`
	DefaultMemoryGB *float32 `json:"default_memory_gb"`
	DefaultDiskGB   *int     `json:"default_disk_gb"`
	Description     *string  `json:"description"`
	URL             *string  `json:"url"`
	Deprecated      *bool    `json:"deprecated"`
	DefaultTag      *string  `json:"default_tag"`
}

// ImportImage creates a new image and optionally a new registry.
func (c *DefaultClient) ImportImage(ctx context.Context, req ImportImageReq) (*Image, error) {
	var img Image
	if err := c.requestBody(ctx, http.MethodPost, "/api/v0/images", req, &img); err != nil {
		return nil, err
	}
	return &img, nil
}

// ImageByID returns an image entity, fetched by its ID.
func (c *DefaultClient) ImageByID(ctx context.Context, id string) (*Image, error) {
	var img Image
	if err := c.requestBody(ctx, http.MethodGet, "/api/v0/images/"+id, nil, &img); err != nil {
		return nil, err
	}
	return &img, nil
}

// OrganizationImages returns all of the images imported for orgID.
func (c *DefaultClient) OrganizationImages(ctx context.Context, orgID string) ([]Image, error) {
	var (
		imgs  []Image
		query = url.Values{}
	)

	query.Set("org", orgID)

	if err := c.requestBody(ctx, http.MethodGet, "/api/v0/images", nil, &imgs, withQueryParams(query)); err != nil {
		return nil, err
	}
	return imgs, nil
}

// UpdateImage applies a partial update to an image resource.
func (c *DefaultClient) UpdateImage(ctx context.Context, imageID string, req UpdateImageReq) error {
	return c.requestBody(ctx, http.MethodPatch, "/api/v0/images/"+imageID, req, nil)
}

// UpdateImageTags refreshes the latest digests for all tags of the image.
func (c *DefaultClient) UpdateImageTags(ctx context.Context, imageID string) error {
	return c.requestBody(ctx, http.MethodPost, "/api/v0/images/"+imageID+"/tags/update", nil, nil)
}
