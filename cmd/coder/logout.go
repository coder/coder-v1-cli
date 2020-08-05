package main

import (
	"os"

	"cdr.dev/coder-cli/internal/config"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	"go.coder.com/flog"
)

func makeLogoutCmd() *cli.Command {
	return &cli.Command{
		Name:   "logout",
		Usage:  "Remove local authentication credentials if any exist",
		Action: logout,
	}
}

func logout(_ *cli.Context) error {
	err := config.Session.Delete()
	if err != nil {
		if os.IsNotExist(err) {
			flog.Info("no active session")
			return nil
		}
		return xerrors.Errorf("delete session: %w", err)
	}
	flog.Success("logged out")
	return nil
}
