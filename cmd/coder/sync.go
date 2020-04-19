package main

import (
	"context"
	"crypto/rand"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/cheggaaa/pb/v3"
	"github.com/spf13/pflag"
	"go.coder.com/cli"
	"go.coder.com/flog"

	"cdr.dev/coder-cli/internal/client"
	"cdr.dev/coder-cli/internal/sync"
	"cdr.dev/coder-cli/wush"
)

type syncCmd struct {
	init      bool
	benchSize int64
}

func (cmd *syncCmd) Spec() cli.CommandSpec {
	return cli.CommandSpec{
		Name:  "sync",
		Usage: "[local directory] [<env name>:<remote directory>]",
		Desc:  "establish a one way directory sync to a remote environment",
	}
}

func (cmd *syncCmd) RegisterFlags(fl *pflag.FlagSet) {
	fl.BoolVarP(&cmd.init, "init", "i", false, "do inititial transfer and exit")
	fl.Int64Var(&cmd.benchSize, "bench", 0, "bench test the wush endpoint")
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

func findEnv(client *client.Client, name string) client.Environment {
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

func (cmd *syncCmd) bench(client *client.Client, env client.Environment) {
	conn, err := client.DialWush(env, nil, "cat")
	if err != nil {
		flog.Fatal("wush failed: %v", err)
	}
	wc := wush.NewClient(context.Background(), conn)
	bar := pb.New64(cmd.benchSize)
	bar.Start()
	go io.Copy(ioutil.Discard, wc.Stdout)
	io.Copy(
		bar.NewProxyWriter(wc.Stdin),
		io.LimitReader(rand.Reader, cmd.benchSize),
	)
	wc.Stdin.Close()
	code, err := wc.Wait()
	if err != nil || code != 0 {
		flog.Error("bench: (code %v) %v", code, err)
	}
}

func (cmd *syncCmd) Run(fl *pflag.FlagSet) {
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
		envName   = remoteTokens[0]
		remoteDir = remoteTokens[1]
	)

	env := findEnv(client, envName)

	if cmd.benchSize > 0 {
		cmd.bench(client, env)
		return
	}

	s := sync.Sync{
		Init:        cmd.init,
		RemoteDir:   remoteDir,
		LocalDir:    local,
		Client:      client,
		Environment: env,
	}
	err = s.Run()
	if err != nil {
		flog.Fatal("sync: %v", err)
	}
}
