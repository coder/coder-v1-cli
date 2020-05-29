package main

import (
	"os"

	"github.com/spf13/pflag"

	"go.coder.com/cli"
	"go.coder.com/flog"

	"cdr.dev/coder-cli/internal/config"
)

type logoutCmd struct {
}

func (cmd logoutCmd) Spec() cli.CommandSpec {
	return cli.CommandSpec{
		Name: "logout",
		Desc: "remote local authentication credentials (if any)",
	}
}

func (cmd logoutCmd) Run(_ *pflag.FlagSet) {
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
