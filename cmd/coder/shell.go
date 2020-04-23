package main

import (
	"context"
	"io"
	"os"
	"os/exec"

	"github.com/mattn/go-isatty"
	"github.com/spf13/pflag"
	"go.coder.com/cli"
	"go.coder.com/flog"

	client "cdr.dev/coder-cli/internal/entclient"
	"cdr.dev/coder-cli/wush"
)

type shellCmd struct {
}

func (cmd *shellCmd) Spec() cli.CommandSpec {
	return cli.CommandSpec{
		Name:    "sh",
		Usage:   "<env name> <command [command args...]>",
		Desc:    "executes a remote command on the environment",
		RawArgs: true,
	}
}

func enableTerminal() {
	out, err := exec.Command("stty", "-f", "/dev/tty",
		"raw",
	).CombinedOutput()
	if err != nil {
		flog.Fatal("configure tty: %v %q", err, out)
	}
}

func (cmd *shellCmd) Run(fl *pflag.FlagSet) {
	if len(fl.Args()) < 2 {
		exitUsage(fl)
	}
	var (
		envName = fl.Arg(0)
		command = fl.Arg(1)
		args    = fl.Args()[2:]
	)

	var (
		entClient = requireAuth()
		env       = findEnv(entClient, envName)
	)

	tty := isatty.IsTerminal(os.Stdout.Fd())
	if tty {
		enableTerminal()
	}

	conn, err := entClient.DialWush(
		env,
		&client.WushOptions{
			TTY:   tty,
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
