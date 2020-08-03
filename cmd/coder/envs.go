package main

import (
	"encoding/json"
	"os"

	"cdr.dev/coder-cli/internal/x/xtabwriter"
	"github.com/urfave/cli"

	"go.coder.com/flog"
)

func makeEnvsCommand() cli.Command {
	var outputFmt string
	return cli.Command{
		Name:        "envs",
		Usage:       "Interact with Coder environments",
		Description: "Perform operations on the Coder environments owned by the active user.",
		Action:      exitHelp,
		Subcommands: []cli.Command{
			{
				Name:        "ls",
				Usage:       "list all environments owned by the active user",
				Description: "List all Coder environments owned by the active user.",
				ArgsUsage:   "[...flags]>",
				Action: func(c *cli.Context) {
					entClient := requireAuth()
					envs := getEnvs(entClient)

					switch outputFmt {
					case "human":
						err := xtabwriter.WriteTable(len(envs), func(i int) interface{} {
							return envs[i]
						})
						requireSuccess(err, "failed to write table: %v", err)
					case "json":
						err := json.NewEncoder(os.Stdout).Encode(envs)
						requireSuccess(err, "failed to write json: %v", err)
					default:
						flog.Fatal("unknown --output value %q", outputFmt)
					}
				},
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:        "output",
						Usage:       "json | human",
						Value:       "human",
						Destination: &outputFmt,
					},
				},
			},
		},
	}
}
