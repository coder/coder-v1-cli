package cmd

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/xerrors"

	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/coder-cli/internal/coderutil"
	"cdr.dev/coder-cli/pkg/clog"
)

// Helpers for working with the Coder API.

// lookupUserOrgs gets a list of orgs the user is apart of.
func lookupUserOrgs(user *coder.User, orgs []coder.Organization) []coder.Organization {
	// NOTE: We don't know in advance how many orgs the user is in so we can't pre-alloc.
	var userOrgs []coder.Organization

	for _, org := range orgs {
		for _, member := range org.Members {
			if member.ID != user.ID {
				continue
			}
			// If we found the user in the org, add it to the list and skip to the next org.
			userOrgs = append(userOrgs, org)
			break
		}
	}
	return userOrgs
}

// getAllWorkspaces gets all workspaces for all users, on all providers.
func getAllWorkspaces(ctx context.Context, client coder.Client) ([]coder.Workspace, error) {
	return client.Workspaces(ctx)
}

// getWorkspaces returns all workspaces for the user.
func getWorkspaces(ctx context.Context, client coder.Client, email string) ([]coder.Workspace, error) {
	user, err := client.UserByEmail(ctx, email)
	if err != nil {
		return nil, xerrors.Errorf("get user: %w", err)
	}

	orgs, err := client.Organizations(ctx)
	if err != nil {
		return nil, xerrors.Errorf("get orgs: %w", err)
	}

	orgs = lookupUserOrgs(user, orgs)

	// NOTE: We don't know in advance how many workspaces we have so we can't pre-alloc.
	var allWorkspaces []coder.Workspace

	for _, org := range orgs {
		workspaces, err := client.UserWorkspacesByOrganization(ctx, user.ID, org.ID)
		if err != nil {
			return nil, xerrors.Errorf("get workspaces for %s: %w", org.Name, err)
		}

		allWorkspaces = append(allWorkspaces, workspaces...)
	}
	return allWorkspaces, nil
}

// searchForWorkspace searches a user's workspaces to find the specified workspaceName. If none is found, the haystack of
// workspace names is returned.
func searchForWorkspace(ctx context.Context, client coder.Client, workspaceName, userEmail string) (_ *coder.Workspace, haystack []string, _ error) {
	workspaces, err := getWorkspaces(ctx, client, userEmail)
	if err != nil {
		return nil, nil, xerrors.Errorf("get workspaces: %w", err)
	}

	// NOTE: We don't know in advance where we will find the workspace, so we can't pre-alloc.
	for _, workspace := range workspaces {
		if workspace.Name == workspaceName {
			return &workspace, nil, nil
		}
		// Keep track of what we found for the logs.
		haystack = append(haystack, workspace.Name)
	}
	return nil, haystack, coder.ErrNotFound
}

// findWorkspace returns a single workspace by name (if it exists.).
func findWorkspace(ctx context.Context, client coder.Client, workspaceName, userEmail string) (*coder.Workspace, error) {
	workspace, haystack, err := searchForWorkspace(ctx, client, workspaceName, userEmail)
	if err != nil {
		return nil, clog.Fatal(
			"failed to find workspace",
			fmt.Sprintf("workspace %q not found in %q", workspaceName, haystack),
			clog.BlankLine,
			clog.Tipf("run \"coder workspaces ls\" to view your workspaces"),
		)
	}
	return workspace, nil
}

type findImgConf struct {
	email   string
	imgName string
	orgName string
}

func findImg(ctx context.Context, client coder.Client, conf findImgConf) (*coder.Image, error) {
	switch {
	case conf.email == "":
		return nil, xerrors.New("user email unset")
	case conf.imgName == "":
		return nil, xerrors.New("image name unset")
	}

	imgs, err := getImgs(ctx, client, getImgsConf{
		email:   conf.email,
		orgName: conf.orgName,
	})
	if err != nil {
		return nil, err
	}

	var possibleMatches []coder.Image

	// The user may provide an image thats not an exact match
	// to one of their imported images but they may be close.
	// We can assist the user by collecting images that contain
	// the user provided image flag value as a substring.
	for _, img := range imgs {
		// If it's an exact match we can just return and exit.
		if img.Repository == conf.imgName {
			return &img, nil
		}
		if strings.Contains(img.Repository, conf.imgName) {
			possibleMatches = append(possibleMatches, img)
		}
	}

	if len(possibleMatches) == 0 {
		return nil, xerrors.New("image not found - did you forget to import this image?")
	}

	lines := []string{clog.Hintf("Did you mean?")}

	for _, img := range possibleMatches {
		lines = append(lines, fmt.Sprintf("  %s", img.Repository))
	}
	return nil, clog.Fatal(
		fmt.Sprintf("image %s not found", conf.imgName),
		lines...,
	)
}

type getImgsConf struct {
	email   string
	orgName string
}

func getImgs(ctx context.Context, client coder.Client, conf getImgsConf) ([]coder.Image, error) {
	u, err := client.UserByEmail(ctx, conf.email)
	if err != nil {
		return nil, err
	}

	orgs, err := client.Organizations(ctx)
	if err != nil {
		return nil, err
	}

	orgs = lookupUserOrgs(u, orgs)

	for _, org := range orgs {
		imgs, err := client.OrganizationImages(ctx, org.ID)
		if err != nil {
			return nil, err
		}
		// If orgName is set we know the user is a multi-org member
		// so we should only return the imported images that beong to the org they specified.
		if conf.orgName != "" && conf.orgName == org.Name {
			return imgs, nil
		}

		if conf.orgName == "" {
			// if orgName is unset we know the user is only part of one org.
			return imgs, nil
		}
	}
	return nil, xerrors.Errorf("org name %q not found", conf.orgName)
}

func isMultiOrgMember(ctx context.Context, client coder.Client, email string) (bool, error) {
	orgs, err := getUserOrgs(ctx, client, email)
	if err != nil {
		return false, err
	}
	return len(orgs) > 1, nil
}

func getUserOrgs(ctx context.Context, client coder.Client, email string) ([]coder.Organization, error) {
	u, err := client.UserByEmail(ctx, email)
	if err != nil {
		return nil, xerrors.New("email not found")
	}

	orgs, err := client.Organizations(ctx)
	if err != nil {
		return nil, xerrors.New("no organizations found")
	}
	return lookupUserOrgs(u, orgs), nil
}

func getWorkspacesByProvider(ctx context.Context, client coder.Client, wpName, userEmail string) ([]coder.Workspace, error) {
	wp, err := coderutil.ProviderByName(ctx, client, wpName)
	if err != nil {
		return nil, err
	}

	workspaces, err := client.WorkspacesByWorkspaceProvider(ctx, wp.ID)
	if err != nil {
		return nil, err
	}

	workspaces, err = filterWorkspacesByUser(ctx, client, userEmail, workspaces)
	if err != nil {
		return nil, err
	}
	return workspaces, nil
}

func filterWorkspacesByUser(ctx context.Context, client coder.Client, userEmail string, workspaces []coder.Workspace) ([]coder.Workspace, error) {
	user, err := client.UserByEmail(ctx, userEmail)
	if err != nil {
		return nil, xerrors.Errorf("get user: %w", err)
	}

	var filteredWorkspaces []coder.Workspace
	for _, workspace := range workspaces {
		if workspace.UserID == user.ID {
			filteredWorkspaces = append(filteredWorkspaces, workspace)
		}
	}
	return filteredWorkspaces, nil
}
