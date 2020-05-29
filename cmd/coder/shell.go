package main

import (
	"context"
	"io"
	"os"
	"os/signal"
	"time"

	"github.com/spf13/pflag"
	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/sys/unix"
	"golang.org/x/time/rate"
	"golang.org/x/xerrors"

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
		Usage:   "<env name> [<command [args...]>]",
		Desc:    "executes a remote command on the environment\nIf no command is specified, the default shell is opened.",
		RawArgs: true,
	}
}

func enableTerminal(fd int) (restore func(), err error) {
	state, err := terminal.MakeRaw(fd)
	if err != nil {
		return restore, xerrors.Errorf("make raw term: %w", err)
	}
	return func() {
		err := terminal.Restore(fd, state)
		if err != nil {
			flog.Error("restore term state: %v", err)
		}
	}, nil
}

func sendResizeEvents(termfd int, client *wush.Client) {
	sigs := make(chan os.Signal, 16)
	signal.Notify(sigs, unix.SIGWINCH)

	// Limit the frequency of resizes to prevent a stuttering effect.
	resizeLimiter := rate.NewLimiter(rate.Every(time.Millisecond*100), 1)

	for {
		width, height, err := terminal.GetSize(termfd)
		if err != nil {
			flog.Error("get term size: %v", err)
			return
		}

		err = client.Resize(width, height)
		if err != nil {
			flog.Error("get term size: %v", err)
			return
		}

		// Do this last so the first resize is sent.
		<-sigs
		resizeLimiter.Wait(context.Background())
	}
}

func (cmd *shellCmd) Run(fl *pflag.FlagSet) {
	if len(fl.Args()) < 1 {
		exitUsage(fl)
	}
	var (
		envName = fl.Arg(0)
		command = fl.Arg(1)
	)

	var args []string
	if command != "" {
		args = fl.Args()[2:]
	}

	// Bring user into shell if no command is specified.
	if command == "" {
		command = "sh"
		args = []string{"-c", "exec $(getent passwd $(whoami) | awk -F: '{ print $7 }')"}
	}

	exitCode, err := runCommand(envName, command, args)
	if err != nil {
		flog.Fatal("run command: %v Is it online?", err)
	}
	os.Exit(exitCode)
}

func runCommand(envName string, command string, args []string) (int, error) {
	var (
		entClient = requireAuth()
		env       = findEnv(entClient, envName)
	)

	termfd := int(os.Stdin.Fd())

	tty := terminal.IsTerminal(termfd)
	if tty {
		restore, err := enableTerminal(termfd)
		if err != nil {
			return -1, err
		}
		defer restore()
	}

	conn, err := entClient.DialWush(
		env,
		&client.WushOptions{
			TTY:   tty,
			Stdin: true,
		}, command, args...)
	if err != nil {
		return -1, err
	}
	ctx := context.Background()

	wc := wush.NewClient(ctx, conn)
	if tty {
		go sendResizeEvents(termfd, wc)
	}

	go func() {
		defer wc.Stdin.Close()
		io.Copy(wc.Stdin, os.Stdin)
	}()
	go io.Copy(os.Stdout, wc.Stdout)
	go io.Copy(os.Stderr, wc.Stderr)

	exitCode, err := wc.Wait()
	if err != nil {
		return -1, err
	}

	return int(exitCode), nil
}
