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
	var orgs []Org
	if err := c.requestBody(ctx, http.MethodGet, "/api/orgs", nil, &orgs); err != nil {
		return nil, err
	}
	return orgs, nil
}
