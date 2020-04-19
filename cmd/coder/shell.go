package main

import (
	"context"
	"io"
	"os"

	"github.com/spf13/pflag"
	"go.coder.com/cli"
	"go.coder.com/flog"

	client "cdr.dev/coder-cli/internal/client"
	"cdr.dev/coder-cli/wush"
)

type shellCmd struct {
}

func (cmd *shellCmd) Spec() cli.CommandSpec {
	return cli.CommandSpec{
		Name:    "sh",
		Usage:   "<env name> -- <command [command args...]>",
		Desc:    "executes a remote command on the environment",
		RawArgs: true,
	}
}

func (cmd *shellCmd) Run(fl *pflag.FlagSet) {
	if len(fl.Args()) < 3 {
		exitUsage(fl)
	}
	var (
		envName = fl.Arg(0)
		_       = fl.Arg(1)
		command = fl.Arg(2)
		args    = fl.Args()[3:]
	)

	entClient := requireAuth()
	env := findEnv(entClient, envName)

	conn, err := entClient.DialWush(
		env,
		&client.WushOptions{
			TTY:   false,
			Stdin: true,
		}, command, args...)
	if err != nil {
		flog.Fatal("dial wush: %v", err)
	}
	ctx := context.Background()

	wc := wush.NewClient(ctx, conn)
	go io.Copy(wc.Stdin, os.Stdin)
	go io.Copy(os.Stdout, wc.Stdout)
	go io.Copy(os.Stderr, wc.Stderr)

	exitCode, err := wc.Wait()
	if err != nil {
		flog.Fatal("wush error: %v", err)
	}
	os.Exit(int(exitCode))
}
