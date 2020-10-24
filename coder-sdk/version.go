package coder

import (
	"context"
	"net/http"
)

// APIVersion parses the coder-version http header from an authenticated request.
func (c Client) APIVersion(ctx context.Context) (string, error) {
	const coderVersionHeaderKey = "coder-version"
	resp, err := c.request(ctx, http.MethodGet, "/api/users/"+Me, nil)
	if err != nil {
		return "", err
	}

	version := resp.Header.Get(coderVersionHeaderKey)
	if version == "" {
		version = "unknown"
	}

	return version, nil
}
