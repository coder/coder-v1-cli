package main

import (
	"os"

	"github.com/spf13/pflag"
)

func exitUsage(fl *pflag.FlagSet) {
	fl.Usage()
	os.Exit(1)
}
