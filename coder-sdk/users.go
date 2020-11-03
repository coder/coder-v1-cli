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

type Role string

type Roles []Role

// Site Roles
const (
	SiteAdmin   Role = "site-admin"
	SiteAuditor Role = "site-auditor"
	SiteManager Role = "site-manager"
	SiteMember  Role = "site-member"
)

type LoginType string

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

// UpdateUserReq defines a modification to the user, updating the
// value of all non-nil values.
type UpdateUserReq struct {
	// TODO(@cmoog) add update password option
	Revoked        *bool      `json:"revoked,omitempty"`
	Roles          *Roles     `json:"roles,omitempty"`
	LoginType      *LoginType `json:"login_type,omitempty"`
	Name           *string    `json:"name,omitempty"`
	Username       *string    `json:"username,omitempty"`
	Email          *string    `json:"email,omitempty"`
	DotfilesGitURL *string    `json:"dotfiles_git_uri,omitempty"`
}

func (c Client) UpdateUser(ctx context.Context, userID string, req UpdateUserReq) error {
	return c.requestBody(ctx, http.MethodPatch, "/api/users/"+userID, req, nil)
}

type CreateUserReq struct {
	Name              string    `json:"name"`
	Username          string    `json:"username"`
	Email             string    `json:"email"`
	Password          string    `json:"password"`
	TemporaryPassword bool      `json:"temporary_password"`
	LoginType         LoginType `json:"login_type"`
	OrganizationsIDs  []string  `json:"organizations"`
}

func (c Client) CreateUser(ctx context.Context, req CreateUserReq) error {
	return c.requestBody(ctx, http.MethodPost, "/api/users", req, nil)
}
