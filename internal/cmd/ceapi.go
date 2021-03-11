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

// Helpers for working with the Coder Enterprise API.

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

// getEnvs returns all environments for the user.
func getEnvs(ctx context.Context, client coder.Client, email string) ([]coder.Environment, error) {
	user, err := client.UserByEmail(ctx, email)
	if err != nil {
		return nil, xerrors.Errorf("get user: %w", err)
	}

	orgs, err := client.Organizations(ctx)
	if err != nil {
		return nil, xerrors.Errorf("get orgs: %w", err)
	}

	orgs = lookupUserOrgs(user, orgs)

	// NOTE: We don't know in advance how many envs we have so we can't pre-alloc.
	var allEnvs []coder.Environment

	for _, org := range orgs {
		envs, err := client.UserEnvironmentsByOrganization(ctx, user.ID, org.ID)
		if err != nil {
			return nil, xerrors.Errorf("get envs for %s: %w", org.Name, err)
		}

		allEnvs = append(allEnvs, envs...)
	}
	return allEnvs, nil
}

// searchForEnv searches a user's environments to find the specified envName. If none is found, the haystack of
// environment names is returned.
func searchForEnv(ctx context.Context, client coder.Client, envName, userEmail string) (_ *coder.Environment, haystack []string, _ error) {
	envs, err := getEnvs(ctx, client, userEmail)
	if err != nil {
		return nil, nil, xerrors.Errorf("get environments: %w", err)
	}

	// NOTE: We don't know in advance where we will find the env, so we can't pre-alloc.
	for _, env := range envs {
		if env.Name == envName {
			return &env, nil, nil
		}
		// Keep track of what we found for the logs.
		haystack = append(haystack, env.Name)
	}
	return nil, haystack, coder.ErrNotFound
}

// findEnv returns a single environment by name (if it exists.).
func findEnv(ctx context.Context, client coder.Client, envName, userEmail string) (*coder.Environment, error) {
	env, haystack, err := searchForEnv(ctx, client, envName, userEmail)
	if err != nil {
		return nil, clog.Fatal(
			"failed to find environment",
			fmt.Sprintf("environment %q not found in %q", envName, haystack),
			clog.BlankLine,
			clog.Tipf("run \"coder envs ls\" to view your environments"),
		)
	}
	return env, nil
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

func getEnvsByProvider(ctx context.Context, client coder.Client, wpName, userEmail string) ([]coder.Environment, error) {
	wp, err := coderutil.ProviderByName(ctx, client, wpName)
	if err != nil {
		return nil, err
	}

	envs, err := client.EnvironmentsByWorkspaceProvider(ctx, wp.ID)
	if err != nil {
		return nil, err
	}

	envs, err = filterEnvsByUser(ctx, client, userEmail, envs)
	if err != nil {
		return nil, err
	}
	return envs, nil
}

func filterEnvsByUser(ctx context.Context, client coder.Client, userEmail string, envs []coder.Environment) ([]coder.Environment, error) {
	user, err := client.UserByEmail(ctx, userEmail)
	if err != nil {
		return nil, xerrors.Errorf("get user: %w", err)
	}

	var filteredEnvs []coder.Environment
	for _, env := range envs {
		if env.UserID == user.ID {
			filteredEnvs = append(filteredEnvs, env)
		}
	}
	return filteredEnvs, nil
}
