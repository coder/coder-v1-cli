package main

import (
	"fmt"

	"cdr.dev/coder-cli/internal/x/xtabwriter"
	"github.com/urfave/cli"
)

func makeEnvsCommand() cli.Command {
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

					w := xtabwriter.NewWriter()
					if len(envs) > 0 {
						_, err := fmt.Fprintln(w, xtabwriter.StructFieldNames(envs[0]))
						requireSuccess(err, "failed to write header: %v", err)
					}
					for _, env := range envs {
						_, err := fmt.Fprintln(w, xtabwriter.StructValues(env))
						requireSuccess(err, "failed to write row: %v", err)
					}
					err := w.Flush()
					requireSuccess(err, "failed to flush tab writer: %v", err)
				},
				Flags: nil,
			},
		},
	}
}
