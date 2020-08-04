package main

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"

	"cdr.dev/coder-cli/internal/x/xterminal"
	"github.com/urfave/cli"

	"go.coder.com/flog"
)

var (
	version string = "unknown"
)

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
	app.Author = "Coder Technologies Inc."
	app.CommandNotFound = func(c *cli.Context, s string) {
		flog.Fatal("command %q not found", s)
	}
	app.Email = "support@coder.com"
	app.Action = exitHelp

	app.Commands = []cli.Command{
		makeLoginCmd(),
		makeLogoutCmd(),
		makeShellCmd(),
		makeUsersCmd(),
		makeConfigSSHCmd(),
		makeSecretsCmd(),
		makeEnvsCommand(),
		makeSyncCmd(),
		makeURLCmd(),
	}
	err = app.Run(os.Args)
	if err != nil {
		flog.Fatal("%v", err)
	}
}

func exitHelp(c *cli.Context) {
	cli.ShowCommandHelpAndExit(c, c.Command.FullName(), 1)
}
