package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
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
	fl.BoolVarP(&cmd.init, "init", "i", false, "do initial transfer and exit")
}

// See https://lxadm.com/Rsync_exit_codes#List_of_standard_rsync_exit_codes.
var IncompatRsync = errors.New("rsync: exit status 2")
var StreamErrRsync = errors.New("rsync: exit status 12")

// Returns local rsync protocol version as a string.
func (s *syncCmd) version() string {
	cmd := exec.Command("rsync", "--version")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	r := bufio.NewReader(stdout)
	if err := cmd.Wait(); err != nil {
		log.Fatal(err)
	}

	versionString := strings.Split(r.ReadLine(), "protocol version ")

	return versionString[1]
}

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
		Init:      cmd.init,
		Env:       env,
		RemoteDir: remoteDir,
		LocalDir:  absLocal,
		Client:    entClient,
	}

	localVersion := s.version()
	remoteVersion, rsyncErr := sync.Version()

	if rsyncErr != nil {
		flog.Info("Unable to determine remote rsync version.  Proceeding cautiously.")
	} else if localVersion != remoteVersion {
		flog.Fatal(fmt.Sprintf("rsync protocol mismatch. local is %s; remote is %s.", localVersion, remoteVersion))
	}

	for err == nil || err == sync.ErrRestartSync {
		err = s.Run()
	}

	if err != nil {
		flog.Fatal("%v", err)
	}
}
