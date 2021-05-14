package coder

import (
	"context"
	"fmt"
	"net/http"
)

// DevURL is the parsed json response record for a devURL from cemanager.
type DevURL struct {
	ID     string `json:"id"     table:"-"`
	URL    string `json:"url"    table:"URL"`
	Port   int    `json:"port"   table:"Port"`
	Access string `json:"access" table:"Access"`
	Name   string `json:"name"   table:"Name"`
	Scheme string `json:"scheme" table:"Scheme"`
}

type delDevURLRequest struct {
	WorkspaceID string `json:"workspace_id"`
	DevURLID    string `json:"url_id"`
}

// DeleteDevURL deletes the specified devurl.
func (c *DefaultClient) DeleteDevURL(ctx context.Context, workspaceID, urlID string) error {
	reqURL := fmt.Sprintf("/api/v0/workspaces/%s/devurls/%s", workspaceID, urlID)

	return c.requestBody(ctx, http.MethodDelete, reqURL, delDevURLRequest{
		WorkspaceID: workspaceID,
		DevURLID:    urlID,
	}, nil)
}

// CreateDevURLReq defines the request parameters for creating a new DevURL.
type CreateDevURLReq struct {
	WorkspaceID string `json:"workspace_id"`
	Port        int    `json:"port"`
	Access      string `json:"access"`
	Name        string `json:"name"`
	Scheme      string `json:"scheme"`
}

// CreateDevURL inserts a new dev URL for the authenticated user.
func (c *DefaultClient) CreateDevURL(ctx context.Context, workspaceID string, req CreateDevURLReq) error {
	return c.requestBody(ctx, http.MethodPost, "/api/v0/workspaces/"+workspaceID+"/devurls", req, nil)
}

// DevURLs fetches the Dev URLs for a given workspace.
func (c *DefaultClient) DevURLs(ctx context.Context, workspaceID string) ([]DevURL, error) {
	var devurls []DevURL
	if err := c.requestBody(ctx, http.MethodGet, "/api/v0/workspaces/"+workspaceID+"/devurls", nil, &devurls); err != nil {
		return nil, err
	}
	return devurls, nil
}

// PutDevURLReq defines the request parameters for overwriting a DevURL.
type PutDevURLReq CreateDevURLReq

// PutDevURL updates an existing devurl for the authenticated user.
func (c *DefaultClient) PutDevURL(ctx context.Context, workspaceID, urlID string, req PutDevURLReq) error {
	return c.requestBody(ctx, http.MethodPut, "/api/v0/workspaces/"+workspaceID+"/devurls/"+urlID, req, nil)
}
