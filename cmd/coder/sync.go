package main

import (
	"bytes"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"cdr.dev/coder-cli/internal/sync"
	"github.com/spf13/cobra"
	"golang.org/x/xerrors"

	"go.coder.com/flog"
)

func makeSyncCmd() *cobra.Command {
	var init bool
	cmd := &cobra.Command{
		Use:   "sync [local directory] [<env name>:<remote directory>]",
		Short: "Establish a one way directory sync to a Coder environment",
		Args:  cobra.ExactArgs(2),
		RunE:  makeRunSync(&init),
	}
	cmd.Flags().BoolVar(&init, "init", false, "do initial transfer and exit")
	return cmd
}

// rsyncVersion returns local rsync protocol version as a string.
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

func makeRunSync(init *bool) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var (
			local  = args[0]
			remote = args[1]
		)

		entClient := requireAuth()

		info, err := os.Stat(local)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return xerrors.Errorf("%s must be a directory", local)
		}

		remoteTokens := strings.SplitN(remote, ":", 2)
		if len(remoteTokens) != 2 {
			flog.Fatal("remote misformatted")
		}
		var (
			envName   = remoteTokens[0]
			remoteDir = remoteTokens[1]
		)

		env, err := findEnv(entClient, envName)
		if err != nil {
			return err
		}

		absLocal, err := filepath.Abs(local)
		if err != nil {
			flog.Fatal("make abs path out of %v: %v", local, absLocal)
		}

		s := sync.Sync{
			Init:      *init,
			Env:       *env,
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
			return err
		}
		return nil
	}
}
