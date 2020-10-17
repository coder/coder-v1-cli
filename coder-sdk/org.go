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
	OrganizationRoles []OrganizationRole `json:"organization_roles"`
	RolesUpdatedAt    time.Time          `json:"roles_updated_at"`
}

// OrganizationRole defines an organization OrganizationRole
type OrganizationRole string

// The OrganizationRole enum values
const (
	RoleOrgMember  OrganizationRole = "organization-member"
	RoleOrgAdmin   OrganizationRole = "organization-admin"
	RoleOrgManager OrganizationRole = "organization-manager"
)

// Organizations gets all Organizations
func (c Client) Organizations(ctx context.Context) ([]Organization, error) {
	var orgs []Organization
	if err := c.requestBody(ctx, http.MethodGet, "/api/orgs", nil, &orgs); err != nil {
		return nil, err
	}
	return orgs, nil
}

// OrgMembers get all members of the given organization
func (c Client) OrgMembers(ctx context.Context, orgID string) ([]OrganizationUser, error) {
	var members []OrganizationUser
	if err := c.requestBody(ctx, http.MethodGet, "/api/orgs/"+orgID+"/members", nil, &members); err != nil {
		return nil, err
	}
	return members, nil
}
