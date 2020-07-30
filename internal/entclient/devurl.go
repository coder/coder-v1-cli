package entclient

import (
	"fmt"
	"net/http"
)

type delDevURLRequest struct {
	EnvID    string `json:"environment_id"`
	DevURLID string `json:"url_id"`
}

// DelDevURL deletes the specified devurl
func (c Client) DelDevURL(envID, urlID string) error {
	reqString := "/api/environments/%s/devurls/%s"
	reqURL := fmt.Sprintf(reqString, envID, urlID)

	res, err := c.request("DELETE", reqURL, delDevURLRequest{
		EnvID:    envID,
		DevURLID: urlID,
	})
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return bodyError(res)
	}

	return nil
}

type createDevURLRequest struct {
	EnvID  string `json:"environment_id"`
	Port   int    `json:"port"`
	Access string `json:"access"`
	Name   string `json:"name"`
}

// InsertDevURL inserts a new devurl for the authenticated user
func (c Client) InsertDevURL(envID string, port int, name, access string) error {
	reqString := "/api/environments/%s/devurls"
	reqURL := fmt.Sprintf(reqString, envID)

	res, err := c.request("POST", reqURL, createDevURLRequest{
		EnvID:  envID,
		Port:   port,
		Access: access,
		Name:   name,
	})
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return bodyError(res)
	}

	return nil
}

type updateDevURLRequest struct {
	EnvID  string `json:"environment_id"`
	Port   int    `json:"port"`
	Access string `json:"access"`
	Name   string `json:"name"`
}

// UpdateDevURL updates an existing devurl for the authenticated user
func (c Client) UpdateDevURL(envID, urlID string, port int, name, access string) error {
	reqString := "/api/environments/%s/devurls/%s"
	reqURL := fmt.Sprintf(reqString, envID, urlID)

	res, err := c.request("PUT", reqURL, updateDevURLRequest{
		EnvID:  envID,
		Port:   port,
		Access: access,
		Name:   name,
	})
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return bodyError(res)
	}

	return nil
}
