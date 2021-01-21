package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/manifoldco/promptui"
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
		Use:   "sh [environment_name] [<command [args...]>]",
		Short: "Open a shell and execute commands in a Coder environment",
		Long: `Execute a remote command on the environment
If no command is specified, the default shell is opened.
If the command is run in an interactive shell, a user prompt will occur if the environment needs to be rebuilt.`,
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

	// Before the command is run, ensure the workspace is on and ready to accept
	// an ssh connection.
	client, err := newClient(ctx)
	if err != nil {
		return err
	}

	env, err := findEnv(ctx, client, envName, coder.Me)
	if err != nil {
		return err
	}

	// TODO: Verify this is the correct behavior
	isInteractive := terminal.IsTerminal(int(os.Stdout.Fd()))
	if isInteractive { // checkAndRebuildEnvironment requires an interactive shell
		// Checks & Rebuilds the environment if needed.
		if err := checkAndRebuildEnvironment(ctx, client, env); err != nil {
			return err
		}
	}

	if err := runCommand(ctx, client, env, command, args); err != nil {
		if exitErr, ok := err.(wsep.ExitError); ok {
			os.Exit(exitErr.Code)
		}
		return xerrors.Errorf("run command: %w", err)
	}
	return nil
}

// rebuildPrompt returns function that prompts the user if they wish to
// rebuild the selected environment if a rebuild is needed. The returned prompt function will
// return an error if the user selects "no".
// This functions returns `nil` if there is no reason to prompt the user to rebuild
// the environment.
func rebuildPrompt(env *coder.Environment) (prompt func() error) {
	// Option 1: If the environment is off, the rebuild is needed
	if env.LatestStat.ContainerStatus == coder.EnvironmentOff {
		confirm := promptui.Prompt{
			Label:     fmt.Sprintf("Environment %q is \"OFF\". Rebuild it now? (this can take several minutes", env.Name),
			IsConfirm: true,
		}
		return func() (err error) {
			_, err = confirm.Run()
			return
		}
	}

	// Option 2: If there are required rebuild messages, the rebuild is needed
	var lines []string
	for _, r := range env.RebuildMessages {
		if r.Required {
			lines = append(lines, clog.Causef(r.Text))
		}
	}

	if len(lines) > 0 {
		confirm := promptui.Prompt{
			Label:     fmt.Sprintf("Environment %q requires a rebuild to work correctly. Do you wish to rebuild it now? (this will take a moment)", env.Name),
			IsConfirm: true,
		}
		// This function also prints the reasons in a log statement.
		// The confirm prompt does not handle new lines well in the label.
		return func() (err error) {
			clog.LogWarn("rebuild required", lines...)
			_, err = confirm.Run()
			return
		}
	}

	// Environment looks good, no need to prompt the user.
	return nil
}

// checkAndRebuildEnvironment will:
//	1. Check if an environment needs to be rebuilt to be used
// 	2. Prompt the user if they want to rebuild the environment (returns an error if they do not)
//	3. Rebuilds the environment and waits for it to be 'ON'
// Conditions for rebuilding are:
//	- Environment is offline
//	- Environment has rebuild messages requiring a rebuild
func checkAndRebuildEnvironment(ctx context.Context, client *coder.Client, env *coder.Environment) error {
	var err error
	rebuildPrompt := rebuildPrompt(env) // Fetch the prompt for rebuilding envs w/ reason

	switch {
	// If this conditonal is true, a rebuild is **required** to make the sh command work.
	case rebuildPrompt != nil:
		// TODO: (@emyrk) I'd like to add a --force and --verbose flags to this command,
		//					but currently DisableFlagParsing is set to true.
		//					To enable force/verbose, we'd have to parse the flags ourselves,
		//					or make the user `coder sh <env> -- [args]`
		//
		if err := rebuildPrompt(); err != nil {
			// User selected not to rebuild :(
			return clog.Fatal(
				"environment is not ready for use",
				"environment requires a rebuild",
				fmt.Sprintf("its current status is %q", env.LatestStat.ContainerStatus),
				clog.BlankLine,
				clog.Tipf("run \"coder envs rebuild %s --follow\" to start the environment", env.Name),
			)
		}

		// Start the rebuild
		if err := client.RebuildEnvironment(ctx, env.ID); err != nil {
			return err
		}

		fallthrough // Fallthrough to watching the logs
	case env.LatestStat.ContainerStatus == coder.EnvironmentCreating:
		// Environment is in the process of being created, just trail the logs
		// and wait until it is done
		clog.LogInfo(fmt.Sprintf("Rebuilding %q", env.Name))

		// Watch the rebuild.
		if err := trailBuildLogs(ctx, client, env.ID); err != nil {
			return err
		}

		// newline after trailBuildLogs to place user on a fresh line for their shell
		fmt.Println()

		// At this point the buildlog is complete, and the status of the env should be 'ON'
		env, err = client.EnvironmentByID(ctx, env.ID)
		if err != nil {
			// If this api call failed, it will likely fail again, no point to retry and make the user wait
			return err
		}

		if env.LatestStat.ContainerStatus != coder.EnvironmentOn {
			// This means we had a timeout
			return clog.Fatal("the environment rebuild ran into an issue",
				fmt.Sprintf("environment %q rebuild has failed and will not come online", env.Name),
				fmt.Sprintf("its current status is %q", env.LatestStat.ContainerStatus),
				clog.BlankLine,
				// TODO: (@emyrk) can they check these logs from the cli? Isn't this the logs that
				//			I just showed them? I'm trying to decide what exactly to tell a user.
				clog.Tipf("take a look at the build logs to determine what went wrong"),
			)
		}

	case env.LatestStat.ContainerStatus == coder.EnvironmentFailed:
		// A failed container might just keep re-failing. I think it should be investigated by the user
		return clog.Fatal("the environment has failed to come online",
			fmt.Sprintf("environment %q is not running", env.Name),
			fmt.Sprintf("its current status is %q", env.LatestStat.ContainerStatus),

			clog.BlankLine,
			clog.Tipf("take a look at the build logs to determine what went wrong"),
			clog.Tipf("run \"coder envs rebuild %s --follow\" to attempt to rebuild the environment", env.Name),
		)
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

func runCommand(ctx context.Context, client *coder.Client, env *coder.Environment, command string, args []string) error {
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
