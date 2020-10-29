package cmd

import (
	"bytes"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/coder-cli/internal/clog"
	"cdr.dev/coder-cli/internal/sync"
	"github.com/spf13/cobra"
	"golang.org/x/xerrors"
)

func syncCmd() *cobra.Command {
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
			ctx    = cmd.Context()
			local  = args[0]
			remote = args[1]
		)

		client, err := newClient()
		if err != nil {
			return err
		}

		remoteTokens := strings.SplitN(remote, ":", 2)
		if len(remoteTokens) != 2 {
			return xerrors.New("remote malformatted")
		}
		var (
			envName   = remoteTokens[0]
			remoteDir = remoteTokens[1]
		)

		env, err := findEnv(cmd.Context(), client, envName, coder.Me)
		if err != nil {
			return err
		}

		info, err := os.Stat(local)
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			return sync.SingleFile(ctx, local, remoteDir, env, client)
		}
		if !info.IsDir() {
			return xerrors.Errorf("local path must lead to a regular file or directory: %w", err)
		}

		absLocal, err := filepath.Abs(local)
		if err != nil {
			return xerrors.Errorf("make abs path out of %s, %s: %w", local, absLocal, err)
		}

		s := sync.Sync{
			Init:      *init,
			Env:       *env,
			RemoteDir: remoteDir,
			LocalDir:  absLocal,
			Client:    client,
		}

		localVersion := rsyncVersion()
		remoteVersion, rsyncErr := s.Version()

		if rsyncErr != nil {
			clog.LogInfo("unable to determine remote rsync version: proceeding cautiously")
		} else if localVersion != remoteVersion {
			return xerrors.Errorf("rsync protocol mismatch: local = %s, remote = %s", localVersion, remoteVersion)
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
