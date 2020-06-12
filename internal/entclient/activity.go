package entclient

import "net/http"

func (c Client) PushActivity(source string, envID string) error {
	res, err := c.request("POST", "/api/metrics/usage/push", map[string]string{
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
