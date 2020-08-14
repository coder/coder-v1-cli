package coder

import (
	"context"
	"net/http"
)

// Org describes an Organization in Coder
type Org struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Members []User `json:"members"`
}

// Orgs gets all Organizations
func (c Client) Orgs(ctx context.Context) ([]Org, error) {
	var os []Org
	err := c.requestBody(ctx, http.MethodGet, "/api/orgs", nil, &os)
	return os, err
}
