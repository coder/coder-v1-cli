package main

import (
	"encoding/json"
	"fmt"
	"os"

	"cdr.dev/coder-cli/internal/xcli"
	"github.com/spf13/pflag"

	"go.coder.com/cli"
)

type usersCmd struct {
}

func (cmd usersCmd) Spec() cli.CommandSpec {
	return cli.CommandSpec{
		Name:  "users",
		Usage: "[subcommand] <flags>",
		Desc:  "interact with user accounts",
	}
}

func (cmd usersCmd) Run(fl *pflag.FlagSet) {
	exitUsage(fl)
}

func (cmd *usersCmd) Subcommands() []cli.Command {
	return []cli.Command{
		&listCmd{},
	}
}

type listCmd struct {
	outputFmt string
}

func (cmd *listCmd) Run(fl *pflag.FlagSet) {
	entClient := requireAuth()

	users, err := entClient.Users()
	xcli.RequireSuccess(err, "failed to get users: %v", err)

	switch cmd.outputFmt {
	case "human":
		w := xcli.HumanReadableWriter()
		if len(users) > 0 {
			_, err = fmt.Fprintln(w, xcli.TabDelimitedStructHeaders(users[0]))
			xcli.RequireSuccess(err, "failed to write: %v", err)
		}
		for _, u := range users {
			_, err = fmt.Fprintln(w, xcli.TabDelimitedStructValues(u))
			xcli.RequireSuccess(err, "failed to write: %v", err)
		}
		err = w.Flush()
		xcli.RequireSuccess(err, "failed to flush writer: %v", err)
	case "json":
		err = json.NewEncoder(os.Stdout).Encode(users)
		xcli.RequireSuccess(err, "failed to encode users to json: %v", err)
	default:
		exitUsage(fl)
	}

}

func (cmd *listCmd) RegisterFlags(fl *pflag.FlagSet) {
	fl.StringVarP(&cmd.outputFmt, "output", "o", "human", "output format (human | json)")
}

func (cmd *listCmd) Spec() cli.CommandSpec {
	return cli.CommandSpec{
		Name:  "ls",
		Usage: "<flags>",
		Desc:  "list all users",
	}
}
