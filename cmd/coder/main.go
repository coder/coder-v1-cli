package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"

	"cdr.dev/coder-cli/internal/x/xterminal"
	"github.com/spf13/pflag"

	"go.coder.com/flog"

	"go.coder.com/cli"
)

var (
	version string = "No version built"
)

type rootCmd struct{}

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
		&envsCmd{},
		&loginCmd{},
		&logoutCmd{},
		&shellCmd{},
		&syncCmd{},
		&urlsCmd{},
		&versionCmd{},
		&configSSHCmd{},
		&usersCmd{},
		&secretsCmd{},
	}
}

func main() {
	if os.Getenv("PPROF") != "" {
		go func() {
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	}

	stdoutState, err := xterminal.MakeOutputRaw(os.Stdout.Fd())
	if err != nil {
		flog.Fatal("failed to set output to raw: %v", err)
	}
	defer xterminal.Restore(os.Stdout.Fd(), stdoutState)

	cli.RunRoot(&rootCmd{})
}

// requireSuccess prints the given message and format args as a fatal error if err != nil
func requireSuccess(err error, msg string, args ...interface{}) {
	if err != nil {
		flog.Fatal(msg, args...)
	}
}
