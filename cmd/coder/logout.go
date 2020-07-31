package main

import (
	"os"

	"cdr.dev/coder-cli/internal/config"
	"github.com/urfave/cli"

	"go.coder.com/flog"
)

func makeLogoutCmd() cli.Command {
	return cli.Command{
		Name: "logout",
		//Usage:       "",
		Description: "remove local authentication credentials (if any)",
		Action:      logout,
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
