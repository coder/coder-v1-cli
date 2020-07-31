package main

import (
	"fmt"

	"github.com/urfave/cli"
)

func makeEnvsCommand() cli.Command {
	return cli.Command{
		Name:        "envs",
		UsageText:   "",
		Description: "interact with Coder environments",
		Subcommands: []cli.Command{
			{
				Name:        "ls",
				Usage:       "list all environments owned by the active user",
				UsageText:   "",
				Description: "",
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
