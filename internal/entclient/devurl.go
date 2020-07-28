package entclient

import (
	"fmt"
	"net/http"
)

// DelDevURL deletes the specified devurl
func (c Client) DelDevURL(envID, urlID string) error {
	reqString := "/api/environments/%s/devurls/%s"
	reqURL := fmt.Sprintf(reqString, envID, urlID)

	res, err := c.request("DELETE", reqURL, map[string]string{
		"environment_id": envID,
		"url_id":         urlID,
	})
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return bodyError(res)
	}

	return nil
}

// UpsertDevURL upserts the specified devurl for the authenticated user
func (c Client) UpsertDevURL(envID, port, access string) error {
	reqString := "/api/environments/%s/devurls"
	reqURL := fmt.Sprintf(reqString, envID)

	res, err := c.request("POST", reqURL, map[string]string{
		"environment_id": envID,
		"port":           port,
		"access":         access,
	})
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return bodyError(res)
	}

	return nil
}
