package cmd

import (
	"context"

	"cdr.dev/coder-cli/coder-sdk"
	"golang.org/x/xerrors"

	"go.coder.com/flog"
)

// Helpers for working with the Coder Enterprise API.

// userOrgs gets a list of orgs the user is apart of.
func userOrgs(user *coder.User, orgs []coder.Org) []coder.Org {
	var uo []coder.Org
outer:
	for _, org := range orgs {
		for _, member := range org.Members {
			if member.ID != user.ID {
				continue
			}
			uo = append(uo, org)
			continue outer
		}
	}
	return uo
}

// getEnvs returns all environments for the user.
func getEnvs(ctx context.Context, client *coder.Client, email string) ([]coder.Environment, error) {
	user, err := client.UserByEmail(ctx, email)
	if err != nil {
		return nil, xerrors.Errorf("get user: %+v", err)
	}

	orgs, err := client.Orgs(ctx)
	if err != nil {
		return nil, xerrors.Errorf("get orgs: %+v", err)
	}

	orgs = userOrgs(user, orgs)

	var allEnvs []coder.Environment

	for _, org := range orgs {
		envs, err := client.EnvironmentsByOrganization(ctx, user.ID, org.ID)
		if err != nil {
			return nil, xerrors.Errorf("get envs for %v: %+v", org.Name, err)
		}

		for _, env := range envs {
			allEnvs = append(allEnvs, env)
		}
	}
	return allEnvs, nil
}

// findEnv returns a single environment by name (if it exists.)
func findEnv(ctx context.Context, client *coder.Client, envName, userEmail string) (*coder.Environment, error) {
	envs, err := getEnvs(ctx, client, userEmail)
	if err != nil {
		return nil, xerrors.Errorf("get environments: %w", err)
	}

	var found []string

	for _, env := range envs {
		found = append(found, env.Name)
		if env.Name == envName {
			return &env, nil
		}
	}
	flog.Error("found %q", found)
	flog.Error("%q not found", envName)
	return nil, xerrors.New("environment not found")
}
