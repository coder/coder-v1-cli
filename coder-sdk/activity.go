package coder

import (
	"context"
	"net/http"
)

type activityRequest struct {
	Source        string `json:"source"`
	EnvironmentID string `json:"environment_id"`
}

// PushActivity pushes CLI activity to Coder.
func (c Client) PushActivity(ctx context.Context, source, envID string) error {
	resp, err := c.request(ctx, http.MethodPost, "/api/metrics/usage/push", activityRequest{
		Source:        source,
		EnvironmentID: envID,
	})
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return bodyError(resp)
	}
	return nil
}
