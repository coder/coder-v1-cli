package coder

import (
	"context"
	"net/http"
	"time"
)

// User describes a Coder user account.
type User struct {
	ID                string    `json:"id"                 table:"-"`
	Email             string    `json:"email"              table:"Email"`
	Username          string    `json:"username"           table:"Username"`
	Name              string    `json:"name"               table:"Name"`
	Roles             []Role    `json:"roles"              table:"-"`
	TemporaryPassword bool      `json:"temporary_password" table:"-"`
	LoginType         string    `json:"login_type"         table:"-"`
	KeyRegeneratedAt  time.Time `json:"key_regenerated_at" table:"-"`
	CreatedAt         time.Time `json:"created_at"         table:"CreatedAt"`
	UpdatedAt         time.Time `json:"updated_at"         table:"-"`
}

// Role defines a Coder Enterprise permissions role group.
type Role string

// Site Roles.
const (
	SiteAdmin   Role = "site-admin"
	SiteAuditor Role = "site-auditor"
	SiteManager Role = "site-manager"
	SiteMember  Role = "site-member"
)

// LoginType defines the enum of valid user login types.
type LoginType string

// LoginType enum options.
const (
	LoginTypeBuiltIn LoginType = "built-in"
	LoginTypeSAML    LoginType = "saml"
	LoginTypeOIDC    LoginType = "oidc"
)

// Me gets the details of the authenticated user.
func (c Client) Me(ctx context.Context) (*User, error) {
	return c.UserByID(ctx, Me)
}

// UserByID get the details of a user by their id.
func (c Client) UserByID(ctx context.Context, id string) (*User, error) {
	var u User
	if err := c.requestBody(ctx, http.MethodGet, "/api/private/users/"+id, nil, &u); err != nil {
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
	if err := c.requestBody(ctx, http.MethodGet, "/api/private/users/me/sshkey", nil, &key); err != nil {
		return nil, err
	}
	return &key, nil
}

// Users gets the list of user accounts.
func (c Client) Users(ctx context.Context) ([]User, error) {
	var u []User
	if err := c.requestBody(ctx, http.MethodGet, "/api/private/users", nil, &u); err != nil {
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

// UpdateUserReq defines a modification to the user, updating the
// value of all non-nil values.
type UpdateUserReq struct {
	// TODO(@cmoog) add update password option
	Revoked        *bool      `json:"revoked,omitempty"`
	Roles          *[]Role    `json:"roles,omitempty"`
	LoginType      *LoginType `json:"login_type,omitempty"`
	Name           *string    `json:"name,omitempty"`
	Username       *string    `json:"username,omitempty"`
	Email          *string    `json:"email,omitempty"`
	DotfilesGitURL *string    `json:"dotfiles_git_uri,omitempty"`
}

// UpdateUser applyes the partial update to the given user.
func (c Client) UpdateUser(ctx context.Context, userID string, req UpdateUserReq) error {
	return c.requestBody(ctx, http.MethodPatch, "/api/private/users/"+userID, req, nil)
}

// CreateUserReq defines the request parameters for creating a new user resource.
type CreateUserReq struct {
	Name              string    `json:"name"`
	Username          string    `json:"username"`
	Email             string    `json:"email"`
	Password          string    `json:"password"`
	TemporaryPassword bool      `json:"temporary_password"`
	LoginType         LoginType `json:"login_type"`
	OrganizationsIDs  []string  `json:"organizations"`
}

// CreateUser creates a new user account.
func (c Client) CreateUser(ctx context.Context, req CreateUserReq) error {
	return c.requestBody(ctx, http.MethodPost, "/api/private/users", req, nil)
}

// DeleteUser deletes a user account.
func (c Client) DeleteUser(ctx context.Context, userID string) error {
	return c.requestBody(ctx, http.MethodDelete, "/api/private/users/"+userID, nil, nil)
}
