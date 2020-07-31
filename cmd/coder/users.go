package main

import (
	"encoding/json"
	"fmt"
	"os"

	"cdr.dev/coder-cli/internal/x/xtabwriter"
	"github.com/urfave/cli"

	"go.coder.com/flog"
)

func makeUsersCmd() cli.Command {
	var output string
	return cli.Command{
		Name:         "users",
		ShortName:    "",
		Aliases:      nil,
		Usage:        "[subcommand] <flags>",
		UsageText:    "",
		Description:  "",
		ArgsUsage:    "",
		Action:       nil,
		OnUsageError: nil,
		Subcommands: []cli.Command{
			{
				Name:        "ls",
				Usage:       "",
				UsageText:   "",
				Description: "",
				ArgsUsage:   "",
				Category:    "",
				Action:      listUsers(&output),
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:        "output",
						Usage:       "",
						Required:    false,
						Value:       "human",
						Destination: &output,
					},
				},
			},
		},
		HelpName: "",
	}
}

func listUsers(outputFmt *string) func(c *cli.Context) {
	return func(c *cli.Context) {
		entClient := requireAuth()

		users, err := entClient.Users()
		requireSuccess(err, "failed to get users: %v", err)

		switch *outputFmt {
		case "human":
			w := xtabwriter.NewWriter()
			if len(users) > 0 {
				_, err = fmt.Fprintln(w, xtabwriter.StructFieldNames(users[0]))
				requireSuccess(err, "failed to write: %v", err)
			}
			for _, u := range users {
				_, err = fmt.Fprintln(w, xtabwriter.StructValues(u))
				requireSuccess(err, "failed to write: %v", err)
			}
			err = w.Flush()
			requireSuccess(err, "failed to flush writer: %v", err)
		case "json":
			err = json.NewEncoder(os.Stdout).Encode(users)
			requireSuccess(err, "failed to encode users to json: %v", err)
		default:
			flog.Fatal("unknown value for --output")
		}
	}
}
