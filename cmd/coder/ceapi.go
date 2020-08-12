package main

import (
	"context"

	"golang.org/x/xerrors"

	"go.coder.com/flog"

	"cdr.dev/coder-cli/internal/entclient"
)

// Helpers for working with the Coder Enterprise API.

// userOrgs gets a list of orgs the user is apart of.
func userOrgs(user *entclient.User, orgs []entclient.Org) []entclient.Org {
	var uo []entclient.Org
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
func getEnvs(ctx context.Context, client *entclient.Client, email string) ([]entclient.Environment, error) {
	user, err := client.UserByEmail(ctx, email)
	if err != nil {
		return nil, xerrors.Errorf("get user: %+v", err)
	}

	orgs, err := client.Orgs(ctx)
	if err != nil {
		return nil, xerrors.Errorf("get orgs: %+v", err)
	}

	orgs = userOrgs(user, orgs)

	var allEnvs []entclient.Environment

	for _, org := range orgs {
		envs, err := client.Envs(ctx, user, org)
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
func findEnv(ctx context.Context, client *entclient.Client, envName, userEmail string) (*entclient.Environment, error) {
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
