package coder

import (
	"context"
	"net/http"
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
	DefaultMemoryGB *int     `json:"default_memory_gb"`
	DefaultDiskGB   *int     `json:"default_disk_gb"`
	Description     *string  `json:"description"`
	URL             *string  `json:"url"`
	Deprecated      *bool    `json:"deprecated"`
	DefaultTag      *string  `json:"default_tag"`
}

// ImportImage creates a new image and optionally a new registry.
func (c Client) ImportImage(ctx context.Context, orgID string, req ImportImageReq) (*Image, error) {
	var img Image
	if err := c.requestBody(ctx, http.MethodPost, "/api/orgs/"+orgID+"/images", req, &img); err != nil {
		return nil, err
	}
	return &img, nil
}

// OrganizationImages returns all of the images imported for orgID.
func (c Client) OrganizationImages(ctx context.Context, orgID string) ([]Image, error) {
	var imgs []Image
	if err := c.requestBody(ctx, http.MethodGet, "/api/orgs/"+orgID+"/images", nil, &imgs); err != nil {
		return nil, err
	}
	return imgs, nil
}

// UpdateImage applies a partial update to an image resource.
func (c Client) UpdateImage(ctx context.Context, imageID string, req UpdateImageReq) error {
	return c.requestBody(ctx, http.MethodPatch, "/api/images/"+imageID, req, nil)
}
