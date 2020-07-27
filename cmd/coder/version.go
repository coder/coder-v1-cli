package main

import (
	"fmt"
	"runtime"

	"github.com/spf13/pflag"

	"go.coder.com/cli"
)

type versionCmd struct{}

func (versionCmd) Spec() cli.CommandSpec {
	return cli.CommandSpec{
		Name:  "version",
		Usage: "",
		Desc:  "print the currently installed CLI version",
	}
}

func (versionCmd) Run(fl *pflag.FlagSet) {
	fmt.Println(
		version,
		runtime.Version(),
		runtime.GOOS+"/"+runtime.GOARCH,
	)
}
