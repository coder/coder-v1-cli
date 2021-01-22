package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/time/rate"
	"golang.org/x/xerrors"
	"nhooyr.io/websocket"

	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/coder-cli/internal/activity"
	"cdr.dev/coder-cli/internal/coderutil"
	"cdr.dev/coder-cli/internal/x/xterminal"
	"cdr.dev/coder-cli/pkg/clog"
	"cdr.dev/wsep"
)

func getEnvsForCompletion(user string) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		ctx := cmd.Context()
		client, err := newClient(ctx)
		if err != nil {
			return nil, cobra.ShellCompDirectiveDefault
		}
		envs, err := getEnvs(ctx, client, user)
		if err != nil {
			return nil, cobra.ShellCompDirectiveDefault
		}

		envNames := make([]string, 0, len(envs))
		for _, e := range envs {
			envNames = append(envNames, e.Name)
		}
		return envNames, cobra.ShellCompDirectiveDefault
	}
}

// special handling for the common case of "coder sh" input without a positional argument.
func shValidArgs(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if err := cobra.MinimumNArgs(1)(cmd, args); err != nil {
		client, err := newClient(ctx)
		if err != nil {
			return clog.Error("missing [environment_name] argument")
		}
		_, haystack, err := searchForEnv(ctx, client, "", coder.Me)
		if err != nil {
			return clog.Error("missing [environment_name] argument",
				fmt.Sprintf("specify one of %q", haystack),
				clog.BlankLine,
				clog.Tipf("run \"coder envs ls\" to view your environments"),
			)
		}
		return clog.Error("missing [environment_name] argument")
	}
	return nil
}

func shCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "sh [environment_name] [<command [args...]>]",
		Short:              "Open a shell and execute commands in a Coder environment",
		Long:               "Execute a remote command on the environment\\nIf no command is specified, the default shell is opened.",
		Args:               shValidArgs,
		DisableFlagParsing: true,
		ValidArgsFunction:  getEnvsForCompletion(coder.Me),
		RunE:               shell,
		Example: `coder sh backend-env
coder sh front-end-dev cat ~/config.json`,
	}
}

func shell(cmd *cobra.Command, cmdArgs []string) error {
	ctx := cmd.Context()
	command := "sh"
	args := []string{"-c"}
	if len(cmdArgs) > 1 {
		args = append(args, strings.Join(cmdArgs[1:], " "))
	} else {
		// Bring user into shell if no command is specified.
		shell := "$(getent passwd $(id -u) | cut -d: -f 7)"
		name := "-$(basename " + shell + ")"
		args = append(args, fmt.Sprintf("exec -a %q %q", name, shell))
	}

	envName := cmdArgs[0]

	if err := runCommand(ctx, envName, command, args); err != nil {
		if exitErr, ok := err.(wsep.ExitError); ok {
			os.Exit(exitErr.Code)
		}
		return xerrors.Errorf("run command: %w", err)
	}
	return nil
}

// sendResizeEvents starts watching for the client's terminal resize signals
// and sends the event to the server so the remote tty can match the client.
func sendResizeEvents(ctx context.Context, termFD uintptr, process wsep.Process) {
	events := xterminal.ResizeEvents(ctx, termFD)

	// Limit the frequency of resizes to prevent a stuttering effect.
	resizeLimiter := rate.NewLimiter(rate.Every(100*time.Millisecond), 1)
	for {
		select {
		case newsize := <-events:
			if err := process.Resize(ctx, newsize.Height, newsize.Width); err != nil {
				return
			}
			_ = resizeLimiter.Wait(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func runCommand(ctx context.Context, envName, command string, args []string) error {
	client, err := newClient(ctx)
	if err != nil {
		return err
	}
	env, err := findEnv(ctx, client, envName, coder.Me)
	if err != nil {
		return xerrors.Errorf("find environment: %w", err)
	}

	// check if a rebuild is required before attempting to open a shell
	for _, r := range env.RebuildMessages {
		// use the first rebuild message that is required
		if r.Required {
			return clog.Error(
				fmt.Sprintf(`environment "%s" requires a rebuild`, env.Name),
				clog.Causef(r.Text), clog.BlankLine,
				clog.Tipf(`run "coder envs rebuild %s" to rebuild`, env.Name),
			)
		}
	}

	termFD := os.Stdout.Fd()

	isInteractive := terminal.IsTerminal(int(termFD))
	if isInteractive {
		// If the client has a tty, take over it by setting the raw mode.
		// This allows for all input to be directly forwarded to the remote process,
		// otherwise, the local terminal would buffer input, interpret special keys, etc.
		stdinState, err := xterminal.MakeRaw(os.Stdin.Fd())
		if err != nil {
			return err
		}
		defer func() {
			// Best effort. If this fails it will result in a broken terminal,
			// but there is nothing we can do about it.
			_ = xterminal.Restore(os.Stdin.Fd(), stdinState)
		}()
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	conn, err := coderutil.DialEnvWsep(ctx, client, env)
	if err != nil {
		return xerrors.Errorf("dial executor: %w", err)
	}
	go heartbeat(ctx, conn, 15*time.Second)

	var cmdEnv []string
	if isInteractive {
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
		TTY:     isInteractive,
		Stdin:   true,
		Env:     cmdEnv,
	})
	if err != nil {
		var closeErr websocket.CloseError
		if xerrors.As(err, &closeErr) {
			return networkErr(env)
		}
		return xerrors.Errorf("start remote command: %w", err)
	}

	// Now that the remote process successfully started, if we have a tty, start the resize event watcher.
	if isInteractive {
		go sendResizeEvents(ctx, termFD, process)
	}

	go func() {
		stdin := process.Stdin()
		defer func() { _ = stdin.Close() }() // Best effort.

		ap := activity.NewPusher(client, env.ID, sshActivityName)
		wr := ap.Writer(stdin)
		if _, err := io.Copy(wr, os.Stdin); err != nil {
			cancel()
		}
	}()
	go func() {
		if _, err := io.Copy(os.Stdout, process.Stdout()); err != nil {
			cancel()
		}
	}()
	go func() {
		if _, err := io.Copy(os.Stderr, process.Stderr()); err != nil {
			cancel()
		}
	}()

	if err := process.Wait(); err != nil {
		var closeErr websocket.CloseError
		if xerrors.Is(err, ctx.Err()) || xerrors.As(err, &closeErr) {
			return networkErr(env)
		}
		return err
	}
	return nil
}

func networkErr(env *coder.Environment) error {
	if env.LatestStat.ContainerStatus != coder.EnvironmentOn {
		return clog.Fatal(
			"environment is not running",
			fmt.Sprintf("environment %q is not running", env.Name),
			fmt.Sprintf("its current status is %q", env.LatestStat.ContainerStatus),
			clog.BlankLine,
			clog.Tipf("run \"coder envs rebuild %s --follow\" to start the environment", env.Name),
		)
	}
	return xerrors.Errorf("network error, is %q online?", env.Name)
}

func heartbeat(ctx context.Context, conn *websocket.Conn, interval time.Duration) {
	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			return
		case <-ticker.C:
			if err := conn.Ping(ctx); err != nil {
				// don't try to do multi-line here because the raw mode makes things weird
				clog.Log(clog.Fatal("failed to ping websocket, exiting: " + err.Error()))
				ticker.Stop()
				os.Exit(1)
			}
		}
	}
}

const sshActivityName = "ssh"
