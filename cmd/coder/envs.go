package main

import (
	"fmt"

	"github.com/urfave/cli"
)

func makeEnvsCommand() cli.Command {
	return cli.Command{
		Name:        "envs",
		Usage:       "Interact with Coder environments",
		Description: "Perform operations on the Coder environments owned by the active user.",
		Subcommands: []cli.Command{
			{
				Name:        "ls",
				Usage:       "list all environments owned by the active user",
				Description: "List all Coder environments owned by the active user.",
				ArgsUsage:   "[...flags]>",
				Action: func(c *cli.Context) {
					entClient := requireAuth()
					envs := getEnvs(entClient)

					for _, env := range envs {
						fmt.Println(env.Name)
					}
				},
				Flags: nil,
			},
		},
	}
}
