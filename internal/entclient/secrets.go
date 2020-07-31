package entclient

import (
	"net/http"
	"time"
)

// Secret describes a Coder secret
type Secret struct {
	ID          string    `json:"id" tab:"-"`
	Name        string    `json:"name"`
	Value       string    `json:"value,omitempty"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" tab:"-"`
}

// Secrets gets all secrets owned by the authed user
func (c *Client) Secrets() ([]Secret, error) {
	var secrets []Secret
	err := c.requestBody(http.MethodGet, "/api/users/me/secrets", nil, &secrets)
	return secrets, err
}

func (c *Client) secretByID(id string) (*Secret, error) {
	var secret Secret
	err := c.requestBody(http.MethodGet, "/api/users/me/secrets/"+id, nil, &secret)
	return &secret, err
}

func (c *Client) secretNameToID(name string) (id string, _ error) {
	secrets, err := c.Secrets()
	if err != nil {
		return "", err
	}
	for _, s := range secrets {
		if s.Name == name {
			return s.ID, nil
		}
	}
	return "", ErrNotFound
}

// SecretByName gets a secret object by name
func (c *Client) SecretByName(name string) (*Secret, error) {
	id, err := c.secretNameToID(name)
	if err != nil {
		return nil, err
	}
	return c.secretByID(id)
}

// InsertSecretReq describes the request body for creating a new secret
type InsertSecretReq struct {
	Name        string `json:"name"`
	Value       string `json:"value"`
	Description string `json:"description"`
}

// InsertSecret adds a new secret for the authed user
func (c *Client) InsertSecret(req InsertSecretReq) error {
	var resp interface{}
	err := c.requestBody(http.MethodPost, "/api/users/me/secrets", req, &resp)
	return err
}

// DeleteSecretByName deletes the authenticated users secret with the given name
func (c *Client) DeleteSecretByName(name string) error {
	id, err := c.secretNameToID(name)
	if err != nil {
		return err
	}
	_, err = c.request(http.MethodDelete, "/api/users/me/secrets/"+id, nil)
	return err
}
