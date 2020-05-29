package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"

	"go.coder.com/cli"
	"go.coder.com/flog"

	"cdr.dev/coder-cli/internal/sync"
)

type syncCmd struct {
	init bool
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
}

// See https://lxadm.com/Rsync_exit_codes#List_of_standard_rsync_exit_codes.
var NoRsync = errors.New("rsync: exit status 2")

func (cmd *syncCmd) Run(fl *pflag.FlagSet) {
	var (
		local  = fl.Arg(0)
		remote = fl.Arg(1)
	)
	if local == "" || remote == "" {
		exitUsage(fl)
	}

	entClient := requireAuth()

	info, err := os.Stat(local)
	if err != nil {
		flog.Fatal("%v", err)
	}
	if !info.IsDir() {
		flog.Fatal("%s must be a directory", local)
	}

	remoteTokens := strings.SplitN(remote, ":", 2)
	if len(remoteTokens) != 2 {
		flog.Fatal("remote misformatted")
	}
	var (
		envName   = remoteTokens[0]
		remoteDir = remoteTokens[1]
	)

	env := findEnv(entClient, envName)

	absLocal, err := filepath.Abs(local)
	if err != nil {
		flog.Fatal("make abs path out of %v: %v", local, absLocal)
	}

	s := sync.Sync{
		Init:        cmd.init,
		Environment: env,
		RemoteDir:   remoteDir,
		LocalDir:    absLocal,
		Client:      entClient,
	}
	for err == nil || err == sync.ErrRestartSync {
		err = s.Run()
	}

	if err == NoRsync {
		flog.Fatal("no compatible rsync present on remote machine")
	} else {
		flog.Fatal("sync: %v", err)
	}
}
