package coder

import (
	"context"
	"net/http"
)

type activityRequest struct {
	Source      string `json:"source"`
	WorkspaceID string `json:"workspace_id"`
}

// PushActivity pushes CLI activity to Coder.
func (c *DefaultClient) PushActivity(ctx context.Context, source, workspaceID string) error {
	resp, err := c.request(ctx, http.MethodPost, "/api/private/metrics/usage/push", activityRequest{
		Source:      source,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return NewHTTPError(resp)
	}
	return nil
}
