package main

import (
	"os"

	"cdr.dev/coder-cli/internal/config"
	"github.com/urfave/cli"

	"go.coder.com/flog"
)

func makeLogoutCmd() cli.Command {
	return cli.Command{
		Name:   "logout",
		Usage:  "Remove local authentication credentials if any exist",
		Action: logout,
	}
}

func logout(c *cli.Context) {
	err := config.Session.Delete()
	if err != nil {
		if os.IsNotExist(err) {
			flog.Info("no active session")
			return
		}
		flog.Fatal("delete session: %v", err)
	}
	flog.Success("logged out")
}
