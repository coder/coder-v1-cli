package main

import (
	"encoding/json"
	"os"

	"cdr.dev/coder-cli/internal/x/xtabwriter"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
)

func makeEnvsCommand() *cli.Command {
	var outputFmt string
	return &cli.Command{
		Name:        "envs",
		Usage:       "Interact with Coder environments",
		Description: "Perform operations on the Coder environments owned by the active user.",
		Action:      exitHelp,
		Subcommands: []*cli.Command{
			{
				Name:        "ls",
				Usage:       "list all environments owned by the active user",
				Description: "List all Coder environments owned by the active user.",
				ArgsUsage:   "[...flags]",
				Action: func(c *cli.Context) error {
					entClient := requireAuth()
					envs, err := getEnvs(entClient)
					if err != nil {
						return err
					}

					switch outputFmt {
					case "human":
						err := xtabwriter.WriteTable(len(envs), func(i int) interface{} {
							return envs[i]
						})
						if err != nil {
							return xerrors.Errorf("write table: %w", err)
						}
					case "json":
						err := json.NewEncoder(os.Stdout).Encode(envs)
						if err != nil {
							return xerrors.Errorf("write environments as JSON: %w", err)
						}
					default:
						return xerrors.Errorf("unknown --output value %q", outputFmt)
					}
					return nil
				},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "output",
						Aliases:     []string{"o"},
						Usage:       "json | human",
						Value:       "human",
						Destination: &outputFmt,
					},
				},
			},
		},
	}
}
