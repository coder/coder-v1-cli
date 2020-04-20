package main

import (
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

func findEnv(client *entclient.Client, name string) entclient.Environment {
	me, err := client.Me()
	if err != nil {
		flog.Fatal("get self: %+v", err)
	}

	orgs, err := client.Orgs()
	if err != nil {
		flog.Fatal("get orgs: %+v", err)
	}

	orgs = userOrgs(me, orgs)

	var found []string

	for _, org := range orgs {
		envs, err := client.Envs(me, org)
		if err != nil {
			flog.Fatal("get envs for %v: %+v", org.Name, err)
		}
		for _, env := range envs {
			found = append(found, env.Name)
			if env.Name != name {
				continue
			}
			return env
		}
	}
	flog.Info("found %q", found)
	flog.Fatal("environment %q not found", name)
	panic("unreachable")
}
