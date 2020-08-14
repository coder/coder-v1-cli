package entclient

import (
	"context"
	"net/http"
)

// PushActivity pushes CLI activity to Coder.
func (c Client) PushActivity(ctx context.Context, source string, envID string) error {
	res, err := c.request(ctx, http.MethodPost, "/api/metrics/usage/push", map[string]string{
		"source":         source,
		"environment_id": envID,
	})
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return bodyError(res)
	}
	return nil
}
