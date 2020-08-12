package entclient

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

// ReqOptions define api request configuration options
type ReqOptions struct {
	// User defines the users whose resources should be targeted
	User string
}

// DefaultReqOptions define reasonable defaults for an api request
var DefaultReqOptions = &ReqOptions{
	User: Me,
}

// Secrets gets all secrets owned by the authed user
func (c *Client) Secrets(ctx context.Context, opts *ReqOptions) ([]Secret, error) {
	if opts == nil {
		opts = DefaultReqOptions
	}
	user, err := c.UserByEmail(ctx, opts.User)
	if err != nil {
		return nil, err
	}
	var secrets []Secret
	err = c.requestBody(ctx, http.MethodGet, "/api/users/"+user.ID+"/secrets", nil, &secrets)
	return secrets, err
}

func (c *Client) secretByID(ctx context.Context, id string, opts *ReqOptions) (*Secret, error) {
	if opts == nil {
		opts = DefaultReqOptions
	}
	user, err := c.UserByEmail(ctx, opts.User)
	if err != nil {
		return nil, err
	}

	var secret Secret
	err = c.requestBody(ctx, http.MethodGet, "/api/users/"+user.ID+"/secrets/"+id, nil, &secret)
	return &secret, err
}

func (c *Client) secretNameToID(ctx context.Context, name string, opts *ReqOptions) (id string, _ error) {
	secrets, err := c.Secrets(ctx, opts)
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
func (c *Client) SecretByName(ctx context.Context, name string, opts *ReqOptions) (*Secret, error) {
	id, err := c.secretNameToID(ctx, name, opts)
	if err != nil {
		return nil, err
	}
	return c.secretByID(ctx, id, opts)
}

// InsertSecretReq describes the request body for creating a new secret
type InsertSecretReq struct {
	Name        string `json:"name"`
	Value       string `json:"value"`
	Description string `json:"description"`
}

// InsertSecret adds a new secret for the authed user
func (c *Client) InsertSecret(ctx context.Context, req InsertSecretReq, opts *ReqOptions) error {
	if opts == nil {
		opts = DefaultReqOptions
	}
	user, err := c.UserByEmail(ctx, opts.User)
	if err != nil {
		return err
	}
	var resp interface{}
	err = c.requestBody(ctx, http.MethodPost, "/api/users/"+user.ID+"/secrets", req, &resp)
	return err
}

// DeleteSecretByName deletes the authenticated users secret with the given name
func (c *Client) DeleteSecretByName(ctx context.Context, name string, opts *ReqOptions) error {
	if opts == nil {
		opts = DefaultReqOptions
	}
	user, err := c.UserByEmail(ctx, opts.User)
	if err != nil {
		return err
	}
	id, err := c.secretNameToID(ctx, name, opts)
	if err != nil {
		return err
	}
	_, err = c.request(ctx, http.MethodDelete, "/api/users/"+user.ID+"/secrets/"+id, nil)
	return err
}
