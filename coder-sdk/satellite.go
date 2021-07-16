package coder

import (
	"context"
	"net/http"
)

type Satellite struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Fingerprint string `json:"fingerprint"`
}

type Satellites struct {
	Data []Satellite `json:"data"`
}

// Satellites fetches all satellitess known to the Coder control plane.
func (c *DefaultClient) Satellites(ctx context.Context) (*Satellites, error) {
	var res Satellites
	err := c.requestBody(ctx, http.MethodGet, "/api/private/satellites", nil, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// CreateSatelliteReq defines the request parameters for creating a new satellite entity.
type CreateSatelliteReq struct {
	Name      string `json:"name"`
	PublicKey string `json:"public_key"`
}

// CreateSatellite creates a new satellite entity.
func (c *DefaultClient) CreateSatellite(ctx context.Context, req CreateSatelliteReq) (*Satellites, error) {
	var res Satellites
	err := c.requestBody(ctx, http.MethodPost, "/api/private/satellites", req, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// DeleteSatelliteByID deletes a satellite entity from the Coder control plane.
func (c *DefaultClient) DeleteSatelliteByID(ctx context.Context, id string) error {
	return c.requestBody(ctx, http.MethodDelete, "/api/private/satellites/"+id, nil, nil)
}
