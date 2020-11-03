package coder

import (
	"context"
	"net/http"
	"time"
)

// Organization describes an Organization in Coder
type Organization struct {
	ID      string             `json:"id"`
	Name    string             `json:"name"`
	Members []OrganizationUser `json:"members"`
}

// OrganizationUser user wraps the basic User type and adds data specific to the user's membership of an organization
type OrganizationUser struct {
	User
	OrganizationRoles []Role    `json:"organization_roles"`
	RolesUpdatedAt    time.Time `json:"roles_updated_at"`
}

// Organization Roles
const (
	RoleOrgMember  Role = "organization-member"
	RoleOrgAdmin   Role = "organization-admin"
	RoleOrgManager Role = "organization-manager"
)

// Organizations gets all Organizations
func (c Client) Organizations(ctx context.Context) ([]Organization, error) {
	var orgs []Organization
	if err := c.requestBody(ctx, http.MethodGet, "/api/orgs", nil, &orgs); err != nil {
		return nil, err
	}
	return orgs, nil
}

func (c Client) OrganizationByID(ctx context.Context, orgID string) (*Organization, error) {
	var org Organization
	err := c.requestBody(ctx, http.MethodGet, "/api/orgs/"+orgID, nil, &org)
	if err != nil {
		return nil, err
	}
	return &org, nil
}

// OrganizationMembers get all members of the given organization
func (c Client) OrganizationMembers(ctx context.Context, orgID string) ([]OrganizationUser, error) {
	var members []OrganizationUser
	if err := c.requestBody(ctx, http.MethodGet, "/api/orgs/"+orgID+"/members", nil, &members); err != nil {
		return nil, err
	}
	return members, nil
}

type UpdateOrganizationReq struct {
	Name                   *string   `json:"name"`
	Description            *string   `json:"description"`
	Default                *bool     `json:"default"`
	AutoOffThreshold       *Duration `json:"auto_off_threshold"`
	CPUProvisioningRate    *float32  `json:"cpu_provisioning_rate"`
	MemoryProvisioningRate *float32  `json:"memory_provisioning_rate"`
}

func (c Client) UpdateOrganization(ctx context.Context, orgID string, req UpdateOrganizationReq) error {
	return c.requestBody(ctx, http.MethodPatch, "/api/orgs/"+orgID, req, nil)
}

type CreateOrganizationReq struct {
	Name                   string   `json:"name"`
	Description            string   `json:"description"`
	Default                bool     `json:"default"`
	ResourceNamespace      string   `json:"resource_namespace"`
	AutoOffThreshold       Duration `json:"auto_off_threshold"`
	CPUProvisioningRate    float32  `json:"cpu_provisioning_rate"`
	MemoryProvisioningRate float32  `json:"memory_provisioning_rate"`
}

func (c Client) CreateOrganization(ctx context.Context, req CreateOrganizationReq) error {
	return c.requestBody(ctx, http.MethodPost, "/api/orgs", req, nil)
}

func (c Client) DeleteOrganization(ctx context.Context, orgID string) error {
	return c.requestBody(ctx, http.MethodDelete, "/api/orgs/"+orgID, nil, nil)
}
