package coder

import (
	"context"
	"net/http"
	"time"
)

// User describes a Coder user account.
type User struct {
	ID        string    `json:"id"         tab:"-"`
	Email     string    `json:"email"      tab:"Email"`
	Username  string    `json:"username"   tab:"Username"`
	Name      string    `json:"name"       tab:"Name"`
	CreatedAt time.Time `json:"created_at" tab:"CreatedAt"`
	UpdatedAt time.Time `json:"updated_at" tab:"-"`
}

// Me gets the details of the authenticated user.
func (c Client) Me(ctx context.Context) (*User, error) {
	return c.UserByID(ctx, Me)
}

// UserByID get the details of a user by their id.
func (c Client) UserByID(ctx context.Context, id string) (*User, error) {
	var u User
	if err := c.requestBody(ctx, http.MethodGet, "/api/users/"+id, nil, &u); err != nil {
		return nil, err
	}
	return &u, nil
}

// SSHKey describes an SSH keypair.
type SSHKey struct {
	PublicKey  string `json:"public_key"`
	PrivateKey string `json:"private_key"`
}

// SSHKey gets the current SSH kepair of the authenticated user.
func (c Client) SSHKey(ctx context.Context) (*SSHKey, error) {
	var key SSHKey
	if err := c.requestBody(ctx, http.MethodGet, "/api/users/me/sshkey", nil, &key); err != nil {
		return nil, err
	}
	return &key, nil
}

// Users gets the list of user accounts.
func (c Client) Users(ctx context.Context) ([]User, error) {
	var u []User
	if err := c.requestBody(ctx, http.MethodGet, "/api/users", nil, &u); err != nil {
		return nil, err
	}
	return u, nil
}

// UserByEmail gets a user by email.
func (c Client) UserByEmail(ctx context.Context, email string) (*User, error) {
	if email == Me {
		return c.Me(ctx)
	}
	users, err := c.Users(ctx)
	if err != nil {
		return nil, err
	}
	for _, u := range users {
		if u.Email == email {
			return &u, nil
		}
	}
	return nil, ErrNotFound
}
