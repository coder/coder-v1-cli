package cmd

import (
	"context"
	"fmt"

	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/coder-cli/internal/clog"
	"golang.org/x/xerrors"
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
func getEnvs(ctx context.Context, client *coder.Client, email string) ([]coder.Environment, error) {
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
		envs, err := client.EnvironmentsByOrganization(ctx, user.ID, org.ID)
		if err != nil {
			return nil, xerrors.Errorf("get envs for %s: %w", org.Name, err)
		}

		allEnvs = append(allEnvs, envs...)
	}
	return allEnvs, nil
}

// findEnv returns a single environment by name (if it exists.)
func findEnv(ctx context.Context, client *coder.Client, envName, userEmail string) (*coder.Environment, error) {
	envs, err := getEnvs(ctx, client, userEmail)
	if err != nil {
		return nil, xerrors.Errorf("get environments: %w", err)
	}

	// NOTE: We don't know in advance where we will find the env, so we can't pre-alloc.
	var found []string
	for _, env := range envs {
		if env.Name == envName {
			return &env, nil
		}
		// Keep track of what we found for the logs.
		found = append(found, env.Name)
	}

	return nil, clog.Fatal(
		"failed to find environment",
		fmt.Sprintf("environment %q not found in %q", envName, found),
		clog.BlankLine,
		clog.Tip("run \"coder envs ls\" to view your environments"),
	)
}

type notFoundButDidFind struct {
	needle   string
	haystack []string
}

func (n notFoundButDidFind) Error() string {
	return fmt.Sprintf("\"%s\" not found in %q: %v", n.needle, n.haystack, coder.ErrNotFound)
}
