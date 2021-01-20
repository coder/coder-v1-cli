package coder

import (
	"context"
	"net/http"
	"time"
)

// ImageTag is a Docker image tag.
type ImageTag struct {
	ImageID           string         `json:"image_id"             table:"-"`
	Tag               string         `json:"tag"                  table:"Tag"`
	LatestHash        string         `json:"latest_hash"          table:"-"`
	HashLastUpdatedAt time.Time      `json:"hash_last_updated_at" table:"-"`
	OSRelease         *OSRelease     `json:"os_release"           table:"OS"`
	Environments      []*Environment `json:"environments"         table:"-"`
	UpdatedAt         time.Time      `json:"updated_at"           table:"UpdatedAt"`
	CreatedAt         time.Time      `json:"created_at"           table:"-"`
}

func (i ImageTag) String() string {
	return i.Tag
}

// OSRelease is the marshalled /etc/os-release file.
type OSRelease struct {
	ID         string `json:"id"`
	PrettyName string `json:"pretty_name"`
	HomeURL    string `json:"home_url"`
}

func (o OSRelease) String() string {
	return o.PrettyName
}

// CreateImageTagReq defines the request parameters for creating a new image tag.
type CreateImageTagReq struct {
	Tag     string `json:"tag"`
	Default bool   `json:"default"`
}

// CreateImageTag creates a new image tag resource.
func (c Client) CreateImageTag(ctx context.Context, imageID string, req CreateImageTagReq) (*ImageTag, error) {
	var tag ImageTag
	if err := c.requestBody(ctx, http.MethodPost, "/api/v0/images/"+imageID+"/tags", req, tag); err != nil {
		return nil, err
	}
	return &tag, nil
}

// DeleteImageTag deletes an image tag resource.
func (c Client) DeleteImageTag(ctx context.Context, imageID, tag string) error {
	return c.requestBody(ctx, http.MethodDelete, "/api/v0/images/"+imageID+"/tags/"+tag, nil, nil)
}

// ImageTags fetch all image tags.
func (c Client) ImageTags(ctx context.Context, imageID string) ([]ImageTag, error) {
	var tags []ImageTag
	if err := c.requestBody(ctx, http.MethodGet, "/api/v0/images/"+imageID+"/tags", nil, &tags); err != nil {
		return nil, err
	}
	return tags, nil
}

// ImageTagByID fetch an image tag by ID.
func (c Client) ImageTagByID(ctx context.Context, imageID, tagID string) (*ImageTag, error) {
	var tag ImageTag
	if err := c.requestBody(ctx, http.MethodGet, "/api/v0/images/"+imageID+"/tags/"+tagID, nil, &tag); err != nil {
		return nil, err
	}
	return &tag, nil
}
