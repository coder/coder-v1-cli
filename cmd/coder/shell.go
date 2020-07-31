package main

import (
	"context"
	"io"
	"os"
	"strings"
	"time"

	"cdr.dev/coder-cli/internal/activity"
	"cdr.dev/coder-cli/internal/x/xterminal"
	"cdr.dev/wsep"
	"github.com/urfave/cli"
	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/time/rate"
	"golang.org/x/xerrors"
	"nhooyr.io/websocket"

	"go.coder.com/flog"
)

func makeShellCmd() cli.Command {
	return cli.Command{
		Name:            "sh",
		Usage:           "Open a shell and execute commands in a Coder environment",
		Description:     "Execute a remote command on the environment\\nIf no command is specified, the default shell is opened.",
		ArgsUsage:       "[env_name] [<command [args...]>]",
		SkipFlagParsing: true,
		SkipArgReorder:  true,
		Action:          shell,
	}
}

func shell(c *cli.Context) {
	var (
		envName = c.Args().First()
		ctx     = context.Background()
	)

	command := "sh"
	args := []string{"-c"}
	if len(c.Args().Tail()) > 0 {
		args = append(args, strings.Join(c.Args().Tail(), " "))
	} else {
		// Bring user into shell if no command is specified.
		args = append(args, "exec $(getent passwd $(whoami) | awk -F: '{ print $7 }')")
	}

	err := runCommand(ctx, envName, command, args)
	if exitErr, ok := err.(wsep.ExitError); ok {
		os.Exit(exitErr.Code)
	}
	if err != nil {
		flog.Fatal("run command: %v", err)
	}
}

func sendResizeEvents(ctx context.Context, termfd uintptr, process wsep.Process) {
	events := xterminal.ResizeEvents(ctx, termfd)

	// Limit the frequency of resizes to prevent a stuttering effect.
	resizeLimiter := rate.NewLimiter(rate.Every(time.Millisecond*100), 1)
	for {
		select {
		case newsize := <-events:
			err := process.Resize(ctx, newsize.Height, newsize.Width)
			if err != nil {
				return
			}
			_ = resizeLimiter.Wait(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func runCommand(ctx context.Context, envName string, command string, args []string) error {
	var (
		entClient = requireAuth()
		env       = findEnv(entClient, envName)
	)

	termfd := os.Stdout.Fd()

	tty := terminal.IsTerminal(int(termfd))
	if tty {
		stdinState, err := xterminal.MakeRaw(os.Stdin.Fd())
		if err != nil {
			return err
		}
		defer xterminal.Restore(os.Stdin.Fd(), stdinState)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	conn, err := entClient.DialWsep(ctx, env)
	if err != nil {
		return err
	}
	go heartbeat(ctx, conn, 15*time.Second)

	var cmdEnv []string
	if tty {
		term := os.Getenv("TERM")
		if term == "" {
			term = "xterm"
		}
		cmdEnv = append(cmdEnv, "TERM="+term)
	}

	execer := wsep.RemoteExecer(conn)
	process, err := execer.Start(ctx, wsep.Command{
		Command: command,
		Args:    args,
		TTY:     tty,
		Stdin:   true,
		Env:     cmdEnv,
	})
	if err != nil {
		return err
	}

	if tty {
		go sendResizeEvents(ctx, termfd, process)
	}

	go func() {
		stdin := process.Stdin()
		defer stdin.Close()

		ap := activity.NewPusher(entClient, env.ID, sshActivityName)
		wr := ap.Writer(stdin)
		_, err := io.Copy(wr, os.Stdin)
		if err != nil {
			cancel()
		}
	}()
	go func() {
		_, err := io.Copy(os.Stdout, process.Stdout())
		if err != nil {
			cancel()
		}
	}()
	go func() {
		_, err := io.Copy(os.Stderr, process.Stderr())
		if err != nil {
			cancel()
		}
	}()
	err = process.Wait()
	if err != nil && xerrors.Is(err, ctx.Err()) {
		return xerrors.Errorf("network error, is %q online?", envName)
	}
	return err
}

func heartbeat(ctx context.Context, c *websocket.Conn, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := c.Ping(ctx)
			if err != nil {
				flog.Error("failed to ping websocket: %v", err)
			}
		}
	}
}

const sshActivityName = "ssh"
