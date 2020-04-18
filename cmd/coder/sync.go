package main

import (
	"os"
	"strings"

	"github.com/spf13/pflag"
	"go.coder.com/cli"
	"go.coder.com/flog"

	"cdr.dev/coder/internal/client"
	"cdr.dev/coder/internal/sync"
)

type syncCmd struct {
}

func (cmd syncCmd) Spec() cli.CommandSpec {
	return cli.CommandSpec{
		Name:  "sync",
		Usage: "[local directory] [<env name>:<remote directory>]",
		Desc:  "establish a one way directory sync to a remote environment",
	}
}

// userOrgs gets a list of orgs the user is apart of.
func userOrgs(user *client.User, orgs []client.Org) []client.Org {
	var uo []client.Org
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

func (cmd syncCmd) findEnv(client *client.Client, name string) client.Environment {
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

//noinspection GoImportUsedAsName
func (cmd syncCmd) Run(fl *pflag.FlagSet) {
	var (
		local  = fl.Arg(0)
		remote = fl.Arg(1)
	)
	if local == "" || remote == "" {
		exitUsage(fl)
	}

	client := requireAuth()

	info, err := os.Stat(local)
	if err != nil {
		flog.Fatal("%v", err)
	}
	if !info.IsDir() {
		flog.Fatal("%s must be a directory", local)
	}

	remoteTokens := strings.SplitN(remote, ":", 2)
	if len(remoteTokens) != 2 {
		flog.Fatal("remote misformmated")
	}
	var (
		envName    = remoteTokens[0]
		remotePAth = remoteTokens[1]
	)

	env := cmd.findEnv(client, envName)
	_ = remotePAth

	s := sync.Sync{
		Client:      client,
		Environment: env,
	}
	err = s.Run()
	if err != nil {
		flog.Fatal("sync: %v", err)
	}
}
