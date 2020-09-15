package coder

import (
	"context"
	"net/http"
	"time"
)

// Secret describes a Coder secret
type Secret struct {
	ID          string    `json:"id"              tab:"-"`
	Name        string    `json:"name"            tab:"Name"`
	Value       string    `json:"value,omitempty" tab:"Value"`
	Description string    `json:"description"     tab:"Description"`
	CreatedAt   time.Time `json:"created_at"      tab:"CreatedAt"`
	UpdatedAt   time.Time `json:"updated_at"      tab:"-"`
}

// Secrets gets all secrets for the given user
func (c *Client) Secrets(ctx context.Context, userID string) ([]Secret, error) {
	var secrets []Secret
	if err := c.requestBody(ctx, http.MethodGet, "/api/users/"+userID+"/secrets", nil, &secrets); err != nil {
		return nil, err
	}
	return secrets, nil
}

// SecretWithValueByName gets the Coder secret with its value by its name.
func (c *Client) SecretWithValueByName(ctx context.Context, name, userID string) (*Secret, error) {
	// Lookup the secret from the name.
	s, err := c.SecretByName(ctx, name, userID)
	if err != nil {
		return nil, err
	}
	// Pull the secret value.
	// NOTE: This is racy, but acceptable. If the secret is gone or the permission changed since we looked up the id,
	//       the call will simply fail and surface the error to the user.
	var secret Secret
	if err := c.requestBody(ctx, http.MethodGet, "/api/users/"+userID+"/secrets/"+s.ID, nil, &secret); err != nil {
		return nil, err
	}
	return &secret, nil
}

// SecretWithValueByID gets the Coder secret with its value by the secret_id.
func (c *Client) SecretWithValueByID(ctx context.Context, id, userID string) (*Secret, error) {
	var secret Secret
	if err := c.requestBody(ctx, http.MethodGet, "/api/users/"+userID+"/secrets/"+id, nil, &secret); err != nil {
		return nil, err
	}
	return &secret, nil
}

// SecretByName gets a secret object by name
func (c *Client) SecretByName(ctx context.Context, name, userID string) (*Secret, error) {
	secrets, err := c.Secrets(ctx, userID)
	if err != nil {
		return nil, err
	}
	for _, s := range secrets {
		if s.Name == name {
			return &s, nil
		}
	}
	return nil, ErrNotFound
}

// InsertSecretReq describes the request body for creating a new secret.
type InsertSecretReq struct {
	Name        string `json:"name"`
	Value       string `json:"value"`
	Description string `json:"description"`
}

// InsertSecret adds a new secret for the authed user
func (c *Client) InsertSecret(ctx context.Context, user *User, req InsertSecretReq) error {
	return c.requestBody(ctx, http.MethodPost, "/api/users/"+user.ID+"/secrets", req, nil)
}

// DeleteSecretByName deletes the authenticated users secret with the given name
func (c *Client) DeleteSecretByName(ctx context.Context, name, userID string) error {
	// Lookup the secret by name to get the ID.
	secret, err := c.SecretByName(ctx, name, userID)
	if err != nil {
		return err
	}
	// Delete the secret.
	// NOTE: This is racy, but acceptable. If the secret is gone or the permission changed since we looked up the id,
	//       the call will simply fail and surface the error to the user.
	if _, err := c.request(ctx, http.MethodDelete, "/api/users/"+userID+"/secrets/"+secret.ID, nil); err != nil {
		return err
	}
	return nil
}
