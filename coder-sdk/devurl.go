package coder

import (
	"context"
	"fmt"
	"net/http"
)

// DevURL is the parsed json response record for a devURL from cemanager.
type DevURL struct {
	ID     string `json:"id"     table:"ID"`
	URL    string `json:"url"    table:"URL"`
	Port   int    `json:"port"   table:"Port"`
	Access string `json:"access" table:"Access"`
	Name   string `json:"name"   table:"Name"`
	Scheme string `json:"scheme" table:"-"`
}

type delDevURLRequest struct {
	EnvID    string `json:"environment_id"`
	DevURLID string `json:"url_id"`
}

// DeleteDevURL deletes the specified devurl.
func (c Client) DeleteDevURL(ctx context.Context, envID, urlID string) error {
	reqURL := fmt.Sprintf("/api/environments/%s/devurls/%s", envID, urlID)

	return c.requestBody(ctx, http.MethodDelete, reqURL, delDevURLRequest{
		EnvID:    envID,
		DevURLID: urlID,
	}, nil)
}

// CreateDevURLReq defines the request parameters for creating a new DevURL.
type CreateDevURLReq struct {
	EnvID  string `json:"environment_id"`
	Port   int    `json:"port"`
	Access string `json:"access"`
	Name   string `json:"name"`
	Scheme string `json:"scheme"`
}

// CreateDevURL inserts a new devurl for the authenticated user.
func (c Client) CreateDevURL(ctx context.Context, envID string, req CreateDevURLReq) error {
	return c.requestBody(ctx, http.MethodPost, "/api/environments/"+envID+"/devurls", req, nil)
}

// PutDevURLReq defines the request parameters for overwriting a DevURL.
type PutDevURLReq CreateDevURLReq

// PutDevURL updates an existing devurl for the authenticated user.
func (c Client) PutDevURL(ctx context.Context, envID, urlID string, req PutDevURLReq) error {
	return c.requestBody(ctx, http.MethodPut, "/api/environments/"+envID+"/devurls/"+urlID, req, nil)
}
