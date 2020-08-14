package coder

import (
	"context"
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

// Secrets gets all secrets for the given user
func (c *Client) Secrets(ctx context.Context, userID string) ([]Secret, error) {
	var secrets []Secret
	err := c.requestBody(ctx, http.MethodGet, "/api/users/"+userID+"/secrets", nil, &secrets)
	return secrets, err
}

// SecretWithValueByName gets the Coder secret with its value by its name.
func (c *Client) SecretWithValueByName(ctx context.Context, name, userID string) (*Secret, error) {
	s, err := c.SecretByName(ctx, name, userID)
	if err != nil {
		return nil, err
	}
	var secret Secret
	err = c.requestBody(ctx, http.MethodGet, "/api/users/"+userID+"/secrets/"+s.ID, nil, &secret)
	return &secret, err
}

// SecretWithValueByID gets the Coder secret with its value by the secret_id.
func (c *Client) SecretWithValueByID(ctx context.Context, id, userID string) (*Secret, error) {
	var secret Secret
	err := c.requestBody(ctx, http.MethodGet, "/api/users/"+userID+"/secrets/"+id, nil, &secret)
	return &secret, err
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

// InsertSecretReq describes the request body for creating a new secret
type InsertSecretReq struct {
	Name        string `json:"name"`
	Value       string `json:"value"`
	Description string `json:"description"`
}

// InsertSecret adds a new secret for the authed user
func (c *Client) InsertSecret(ctx context.Context, user *User, req InsertSecretReq) error {
	var resp interface{}
	return c.requestBody(ctx, http.MethodPost, "/api/users/"+user.ID+"/secrets", req, &resp)
}

// DeleteSecretByName deletes the authenticated users secret with the given name
func (c *Client) DeleteSecretByName(ctx context.Context, name, userID string) error {
	secret, err := c.SecretByName(ctx, name, userID)
	if err != nil {
		return err
	}
	_, err = c.request(ctx, http.MethodDelete, "/api/users/"+userID+"/secrets/"+secret.ID, nil)
	return err
}
