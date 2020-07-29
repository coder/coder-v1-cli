package main

import (
	"fmt"

	"github.com/spf13/pflag"

	"go.coder.com/cli"
)

type envsCmd struct {
}

func (cmd envsCmd) Spec() cli.CommandSpec {
	return cli.CommandSpec{
		Name: "envs",
		Desc: "get a list of environments owned by the authenticated user",
	}
}

func (cmd envsCmd) Run(fl *pflag.FlagSet) {
	entClient := requireAuth()

	envs := getEnvs(entClient)

	for _, env := range envs {
		fmt.Println(env.Name)
	}
}
