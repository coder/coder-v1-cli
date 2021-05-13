package cmd

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"

	"github.com/spf13/cobra"
	"golang.org/x/term"
	"golang.org/x/xerrors"

	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/coder-cli/pkg/clog"
)

var (
	showInteractiveOutput = term.IsTerminal(int(os.Stdout.Fd()))
)

func sshCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "ssh [workspace_name] [<command [args...]>]",
		Short: "Enter a shell of execute a command over SSH into a Coder workspace",
		Args:  shValidArgs,
		Example: `coder ssh my-dev
coder ssh my-dev pwd`,
		Aliases:               []string{"sh"},
		DisableFlagParsing:    true,
		DisableFlagsInUseLine: true,
		RunE:                  shell,
	}
	return &cmd
}

func shell(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	client, err := newClient(ctx, true)
	if err != nil {
		return err
	}
	me, err := client.Me(ctx)
	if err != nil {
		return err
	}
	workspace, err := findWorkspace(ctx, client, args[0], coder.Me)
	if err != nil {
		return err
	}
	if workspace.LatestStat.ContainerStatus != coder.WorkspaceOn {
		return clog.Error("workspace not available",
			fmt.Sprintf("current status: \"%s\"", workspace.LatestStat.ContainerStatus),
			clog.BlankLine,
			clog.Tipf("use \"coder ws rebuild %s\" to rebuild this workspace", workspace.Name),
		)
	}
	wp, err := client.WorkspaceProviderByID(ctx, workspace.ResourcePoolID)
	if err != nil {
		return err
	}
	u, err := url.Parse(wp.EnvproxyAccessURL)
	if err != nil {
		return err
	}

	usr, err := user.Current()
	if err != nil {
		return xerrors.Errorf("get user home directory: %w", err)
	}
	privateKeyFilepath := filepath.Join(usr.HomeDir, ".ssh", "coder_enterprise")

	err = writeSSHKey(ctx, client, privateKeyFilepath)
	if err != nil {
		return err
	}
	ssh := exec.CommandContext(ctx,
		"ssh", "-i"+privateKeyFilepath,
		fmt.Sprintf("%s-%s@%s", me.Username, workspace.Name, u.Hostname()),
	)
	if len(args) > 1 {
		ssh.Args = append(ssh.Args, args[1:]...)
	}
	ssh.Stderr = os.Stderr
	ssh.Stdout = os.Stdout
	ssh.Stdin = os.Stdin
	err = ssh.Run()
	var exitErr *exec.ExitError
	if xerrors.As(err, &exitErr) {
		os.Exit(exitErr.ExitCode())
		return xerrors.New("unreachable")
	}
	return err
}

// special handling for the common case of "coder sh" input without a positional argument.
func shValidArgs(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	err := cobra.MinimumNArgs(1)(cmd, args)
	if err != nil {
		client, err := newClient(ctx, true)
		if err != nil {
			return clog.Error("missing [workspace_name] argument")
		}
		_, haystack, err := searchForWorkspace(ctx, client, "", coder.Me)
		if err != nil {
			return clog.Error("missing [workspace_name] argument",
				fmt.Sprintf("specify one of %q", haystack),
				clog.BlankLine,
				clog.Tipf("run \"coder ws ls\" to view your workspaces"),
			)
		}
		return clog.Error("missing [workspace_name] argument")
	}
	return nil
}
