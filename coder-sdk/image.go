package coder

import (
	"context"
	"net/http"
)

// Image describes a Coder Image.
type Image struct {
	ID              string  `json:"id"`
	OrganizationID  string  `json:"organization_id"`
	Repository      string  `json:"repository"`
	Description     string  `json:"description"`
	URL             string  `json:"url"` // User-supplied URL for image.
	DefaultCPUCores float32 `json:"default_cpu_cores"`
	DefaultMemoryGB float32 `json:"default_memory_gb"`
	DefaultDiskGB   int     `json:"default_disk_gb"`
	Deprecated      bool    `json:"deprecated"`
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
