package coder

import (
	"context"
	"fmt"
	"net/http"
)

// UpdateLastConnectionAt updates the last connection at attribute of a workspace.
func (c *DefaultClient) UpdateLastConnectionAt(ctx context.Context, workspaceID string) error {
	reqURL := fmt.Sprintf("/api/private/envagent/%s/update-last-connection-at", workspaceID)
	resp, err := c.request(ctx, http.MethodPost, reqURL, nil)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return NewHTTPError(resp)
	}

	return nil
}
