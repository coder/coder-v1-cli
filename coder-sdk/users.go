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
func (c *DefaultClient) Me(ctx context.Context) (*User, error) {
	return c.UserByID(ctx, Me)
}

// UserByID get the details of a user by their id.
func (c *DefaultClient) UserByID(ctx context.Context, id string) (*User, error) {
	var u User
	if err := c.requestBody(ctx, http.MethodGet, "/api/v0/users/"+id, nil, &u); err != nil {
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
func (c *DefaultClient) SSHKey(ctx context.Context) (*SSHKey, error) {
	var key SSHKey
	if err := c.requestBody(ctx, http.MethodGet, "/api/v0/users/me/sshkey", nil, &key); err != nil {
		return nil, err
	}
	return &key, nil
}

// Users gets the list of user accounts.
func (c *DefaultClient) Users(ctx context.Context) ([]User, error) {
	var u []User
	if err := c.requestBody(ctx, http.MethodGet, "/api/v0/users", nil, &u); err != nil {
		return nil, err
	}
	return u, nil
}

// UserByEmail gets a user by email.
func (c *DefaultClient) UserByEmail(ctx context.Context, email string) (*User, error) {
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
	*UserPasswordSettings
	Revoked        *bool      `json:"revoked,omitempty"`
	Roles          *[]Role    `json:"roles,omitempty"`
	LoginType      *LoginType `json:"login_type,omitempty"`
	Name           *string    `json:"name,omitempty"`
	Username       *string    `json:"username,omitempty"`
	Email          *string    `json:"email,omitempty"`
	DotfilesGitURL *string    `json:"dotfiles_git_uri,omitempty"`
}

// UserPasswordSettings allows modification of the user's password
// settings.
//
// These settings are only applicable to users managed using the
// built-in authentication provider; users authenticating using
// OAuth must change their password through the identity provider
// instead.
type UserPasswordSettings struct {
	// OldPassword is the account's current password.
	OldPassword string `json:"old_password,omitempty"`

	// Password is the new password, which may be a temporary password.
	Password string `json:"password,omitempty"`

	// Temporary indicates that API access should be restricted to the
	// password change API and a few other APIs. If set to true, Coder
	// will prompt the user to change their password upon their next
	// login through the web interface.
	Temporary bool `json:"temporary_password,omitempty"`
}

// UpdateUser applyes the partial update to the given user.
func (c *DefaultClient) UpdateUser(ctx context.Context, userID string, req UpdateUserReq) error {
	return c.requestBody(ctx, http.MethodPatch, "/api/v0/users/"+userID, req, nil)
}

// UpdateUXState applies a partial update of the user's UX State.
func (c *DefaultClient) UpdateUXState(ctx context.Context, userID string, uxsPartial map[string]interface{}) error {
	if err := c.requestBody(ctx, http.MethodPut, "/api/private/users/"+userID+"/ux-state", uxsPartial, nil); err != nil {
		return err
	}
	return nil
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
func (c *DefaultClient) CreateUser(ctx context.Context, req CreateUserReq) error {
	return c.requestBody(ctx, http.MethodPost, "/api/v0/users", req, nil)
}

// DeleteUser deletes a user account.
func (c *DefaultClient) DeleteUser(ctx context.Context, userID string) error {
	return c.requestBody(ctx, http.MethodDelete, "/api/v0/users/"+userID, nil, nil)
}
