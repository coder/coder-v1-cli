package main

import (
	"bytes"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"cdr.dev/coder-cli/internal/sync"
	"github.com/urfave/cli"
	"golang.org/x/xerrors"

	"go.coder.com/flog"
)

func makeSyncCmd() cli.Command {
	var init bool
	return cli.Command{
		Name:      "sync",
		Usage:     "Establish a one way directory sync to a Coder environment",
		ArgsUsage: "[local directory] [<env name>:<remote directory>]",
		Before: func(c *cli.Context) error {
			if c.Args().Get(0) == "" || c.Args().Get(1) == "" {
				return xerrors.Errorf("[local] and [remote] arguments are required")
			}
			return nil
		},
		Action: makeRunSync(&init),
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:        "init",
				Usage:       "do initial transfer and exit",
				Destination: &init,
			},
		},
	}
}

// version returns local rsync protocol version as a string.
func rsyncVersion() string {
	cmd := exec.Command("rsync", "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}

	firstLine, err := bytes.NewBuffer(out).ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}
	versionString := strings.Split(firstLine, "protocol version ")

	return versionString[1]
}

func makeRunSync(init *bool) func(c *cli.Context) {
	return func(c *cli.Context) {
		var (
			local  = c.Args().Get(0)
			remote = c.Args().Get(1)
		)

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
			Init:      *init,
			Env:       env,
			RemoteDir: remoteDir,
			LocalDir:  absLocal,
			Client:    entClient,
		}

		localVersion := rsyncVersion()
		remoteVersion, rsyncErr := s.Version()

		if rsyncErr != nil {
			flog.Info("Unable to determine remote rsync version.  Proceeding cautiously.")
		} else if localVersion != remoteVersion {
			flog.Fatal("rsync protocol mismatch: local = %v, remote = %v", localVersion, rsyncErr)
		}

		for err == nil || err == sync.ErrRestartSync {
			err = s.Run()
		}
		if err != nil {
			flog.Fatal("%v", err)
		}
	}
}
