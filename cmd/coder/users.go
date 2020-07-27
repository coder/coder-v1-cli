package main

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"text/tabwriter"

	"github.com/spf13/pflag"

	"go.coder.com/cli"
	"go.coder.com/flog"
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

func tabDelimited(data interface{}) string {
	v := reflect.ValueOf(data)
	s := &strings.Builder{}
	for i := 0; i < v.NumField(); i++ {
		s.WriteString(fmt.Sprintf("%s\t", v.Field(i).Interface()))
	}
	return s.String()
}

func (cmd *listCmd) Run(fl *pflag.FlagSet) {
	entClient := requireAuth()

	users, err := entClient.Users()
	if err != nil {
		flog.Fatal("failed to get users: %v", err)
	}

	switch cmd.outputFmt {
	case "human":
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		for _, u := range users {
			_, err = fmt.Fprintln(w, tabDelimited(u))
			if err != nil {
				flog.Fatal("failed to write: %v", err)
			}
		}
		err = w.Flush()
		if err != nil {
			flog.Fatal("failed to flush writer: %v", err)
		}
	case "json":
		err = json.NewEncoder(os.Stdout).Encode(users)
		if err != nil {
			flog.Fatal("failed to encode users to json: %v", err)
		}
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
