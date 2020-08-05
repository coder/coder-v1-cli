package main

import (
	"encoding/json"
	"os"

	"cdr.dev/coder-cli/internal/x/xtabwriter"
	"github.com/urfave/cli"
	"golang.org/x/xerrors"
)

func makeUsersCmd() cli.Command {
	var output string
	return cli.Command{
		Name:   "users",
		Usage:  "Interact with Coder user accounts",
		Action: exitHelp,
		Subcommands: []cli.Command{
			{
				Name:   "ls",
				Usage:  "list all user accounts",
				Action: listUsers(&output),
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:        "output",
						Usage:       "json | human",
						Value:       "human",
						Destination: &output,
					},
				},
			},
		},
	}
}

func listUsers(outputFmt *string) func(c *cli.Context) error {
	return func(c *cli.Context) error {
		entClient := requireAuth()

		users, err := entClient.Users()
		if err != nil {
			return xerrors.Errorf("get users: %w", err)
		}

		switch *outputFmt {
		case "human":
			err := xtabwriter.WriteTable(len(users), func(i int) interface{} {
				return users[i]
			})
			if err != nil {
				return xerrors.Errorf("write table: %w", err)
			}
		case "json":
			err = json.NewEncoder(os.Stdout).Encode(users)
			if err != nil {
				return xerrors.Errorf("encode users as json: %w", err)
			}
		default:
			return xerrors.New("unknown value for --output")
		}
		return nil
	}
}
