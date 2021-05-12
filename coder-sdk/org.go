package coder

import (
	"context"
	"net/http"
	"time"
)

// Organization describes an Organization in Coder.
type Organization struct {
	ID                     string             `json:"id"`
	Name                   string             `json:"name"`
	Description            string             `json:"description"`
	Default                bool               `json:"default"`
	Members                []OrganizationUser `json:"members"`
	WorkspaceCount         int                `json:"workspace_count"`
	ResourceNamespace      string             `json:"resource_namespace"`
	CreatedAt              time.Time          `json:"created_at"`
	UpdatedAt              time.Time          `json:"updated_at"`
	AutoOffThreshold       Duration           `json:"auto_off_threshold"`
	CPUProvisioningRate    float32            `json:"cpu_provisioning_rate"`
	MemoryProvisioningRate float32            `json:"memory_provisioning_rate"`
}

// OrganizationUser user wraps the basic User type and adds data specific to the user's membership of an organization.
type OrganizationUser struct {
	User
	OrganizationRoles []Role    `json:"organization_roles"`
	RolesUpdatedAt    time.Time `json:"roles_updated_at"`
}

// Organization Roles.
const (
	RoleOrgMember  Role = "organization-member"
	RoleOrgAdmin   Role = "organization-admin"
	RoleOrgManager Role = "organization-manager"
)

// Organizations gets all Organizations.
func (c *DefaultClient) Organizations(ctx context.Context) ([]Organization, error) {
	var orgs []Organization
	if err := c.requestBody(ctx, http.MethodGet, "/api/v0/orgs", nil, &orgs); err != nil {
		return nil, err
	}
	return orgs, nil
}

// OrganizationByID get the Organization by its ID.
func (c *DefaultClient) OrganizationByID(ctx context.Context, orgID string) (*Organization, error) {
	var org Organization
	err := c.requestBody(ctx, http.MethodGet, "/api/v0/orgs/"+orgID, nil, &org)
	if err != nil {
		return nil, err
	}
	return &org, nil
}

// OrganizationMembers get all members of the given organization.
func (c *DefaultClient) OrganizationMembers(ctx context.Context, orgID string) ([]OrganizationUser, error) {
	var members []OrganizationUser
	if err := c.requestBody(ctx, http.MethodGet, "/api/v0/orgs/"+orgID+"/members", nil, &members); err != nil {
		return nil, err
	}
	return members, nil
}

// UpdateOrganizationReq describes the patch request parameters to provide partial updates to an Organization resource.
type UpdateOrganizationReq struct {
	Name                   *string   `json:"name"`
	Description            *string   `json:"description"`
	Default                *bool     `json:"default"`
	AutoOffThreshold       *Duration `json:"auto_off_threshold"`
	CPUProvisioningRate    *float32  `json:"cpu_provisioning_rate"`
	MemoryProvisioningRate *float32  `json:"memory_provisioning_rate"`
}

// UpdateOrganization applys a partial update of an Organization resource.
func (c *DefaultClient) UpdateOrganization(ctx context.Context, orgID string, req UpdateOrganizationReq) error {
	return c.requestBody(ctx, http.MethodPatch, "/api/v0/orgs/"+orgID, req, nil)
}

// CreateOrganizationReq describes the request parameters to create a new Organization.
type CreateOrganizationReq struct {
	Name                   string   `json:"name"`
	Description            string   `json:"description"`
	Default                bool     `json:"default"`
	ResourceNamespace      string   `json:"resource_namespace"`
	AutoOffThreshold       Duration `json:"auto_off_threshold"`
	CPUProvisioningRate    float32  `json:"cpu_provisioning_rate"`
	MemoryProvisioningRate float32  `json:"memory_provisioning_rate"`
}

// CreateOrganization creates a new Organization in Coder.
func (c *DefaultClient) CreateOrganization(ctx context.Context, req CreateOrganizationReq) error {
	return c.requestBody(ctx, http.MethodPost, "/api/v0/orgs", req, nil)
}

// DeleteOrganization deletes an organization.
func (c *DefaultClient) DeleteOrganization(ctx context.Context, orgID string) error {
	return c.requestBody(ctx, http.MethodDelete, "/api/v0/orgs/"+orgID, nil, nil)
}
