package main

import (
	"os"

	"github.com/spf13/pflag"
	"go.coder.com/flog"
)

func exitOnError(err error) {
	if err != nil {
		flog.Fatal("%+v", err.Error())
	}
}

func exitAfter(err error) {
	exitOnError(err)
	os.Exit(0)
}

func exitUsage(fl *pflag.FlagSet) {
	fl.Usage()
	os.Exit(1)
}
