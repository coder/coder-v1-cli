package entclient

import (
	"fmt"
	"net/http"
)

func (c Client) DelDevURL(envID, urlID string) error {
	reqString := "/api/environments/%s/devurls/%s"
	reqUrl := fmt.Sprintf(reqString, envID, urlID)

	res, err := c.request("DELETE", reqUrl, map[string]string{
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

func (c Client) UpsertDevURL(envID, port, access string) error {
	reqString := "/api/environments/%s/devurls"
	reqUrl := fmt.Sprintf(reqString, envID)

	res, err := c.request("POST", reqUrl, map[string]string{
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
