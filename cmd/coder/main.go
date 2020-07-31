package main

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"

	"cdr.dev/coder-cli/internal/x/xterminal"
	"github.com/spf13/pflag"
	"github.com/urfave/cli"

	cdrcli "go.coder.com/cli"
	"go.coder.com/flog"
)

var (
	version string = "unknown"
)

type rootCmd struct{}

func (r *rootCmd) Run(fl *pflag.FlagSet) {

	fl.Usage()
}

func (r *rootCmd) Spec() cdrcli.CommandSpec {
	return cdrcli.CommandSpec{
		Name:  "coder",
		Usage: "[subcommand] [flags]",
		Desc:  "coder provides a CLI for working with an existing Coder Enterprise installation.",
	}
}

func (r *rootCmd) Subcommands() []cdrcli.Command {
	return []cdrcli.Command{
		&envsCmd{},
		&syncCmd{},
		&urlsCmd{},
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

	app := cli.NewApp()
	app.Name = "coder"
	app.Usage = "coder provides a CLI for working with an existing Coder Enterprise installation"
	app.Version = fmt.Sprintf("%s %s %s/%s", version, runtime.Version(), runtime.GOOS, runtime.GOARCH)
	app.Commands = []cli.Command{
		makeLoginCmd(),
		makeLogoutCmd(),
		makeShellCmd(),
		makeUsersCmd(),
		makeConfigSSHCmd(),
	}
	err = app.Run(os.Args)
	if err != nil {
		flog.Fatal("%v", err)
	}
}

// requireSuccess prints the given message and format args as a fatal error if err != nil
func requireSuccess(err error, msg string, args ...interface{}) {
	if err != nil {
		flog.Fatal(msg, args...)
	}
}
