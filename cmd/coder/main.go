package main

import (
	"github.com/spf13/pflag"
	"go.coder.com/cli"
	"log"
	"os"
	"net/http"
	_ "net/http/pprof"
)

type rootCmd struct {
}

func (r *rootCmd) Run(fl *pflag.FlagSet) {
	fl.Usage()
}

func (r *rootCmd) Spec() cli.CommandSpec {
	return cli.CommandSpec{
		Name:  "coder",
		Usage: "[subcommand] [flags]",
		Desc:  "coder provides a CLI for working with an existing Coder Enterprise installation.",
	}
}

func (r *rootCmd) Subcommands() []cli.Command {
	return []cli.Command{
		loginCmd{},
		logoutCmd{},
		&shellCmd{},
		&syncCmd{},
	}
}

func main() {
	if os.Getenv("PPROF") != "" {
		go func() {
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	}
	cli.RunRoot(&rootCmd{})
}
