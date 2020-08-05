package main

import (
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
func getEnvs(client *entclient.Client) ([]entclient.Environment, error) {
	me, err := client.Me()
	if err != nil {
		return nil, xerrors.Errorf("get self: %+v", err)
	}

	orgs, err := client.Orgs()
	if err != nil {
		return nil, xerrors.Errorf("get orgs: %+v", err)
	}

	orgs = userOrgs(me, orgs)

	var allEnvs []entclient.Environment

	for _, org := range orgs {
		envs, err := client.Envs(me, org)
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
func findEnv(client *entclient.Client, name string) (*entclient.Environment, error) {
	envs, err := getEnvs(client)
	if err != nil {
		return nil, xerrors.Errorf("get environments: %w", err)
	}

	var found []string

	for _, env := range envs {
		found = append(found, env.Name)
		if env.Name == name {
			return &env, nil
		}
	}
	flog.Error("found %q", found)
	flog.Error("%q not found", name)
	return nil, xerrors.New("environment not found")
}
