package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"golang.org/x/term"
	"golang.org/x/xerrors"

	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/coder-cli/internal/ssh"
	"cdr.dev/coder-cli/pkg/clog"
)

var (
	showInteractiveOutput = term.IsTerminal(int(os.Stdout.Fd()))
)

func sshCmd() *cobra.Command {
	var configpath string

	cmd := &cobra.Command{
		Use:   "ssh [workspace_name] [<command [args...]>]",
		Short: "Enter a shell of execute a command over SSH into a Coder workspace",
		Args:  shValidArgs,
		Example: `coder ssh my-dev
coder ssh my-dev pwd`,
		Aliases:               []string{"sh"},
		DisableFlagParsing:    true,
		DisableFlagsInUseLine: true,
		RunE:                  shell(&configpath),
	}

	cmd.Flags().StringVar(&configpath, "filepath", ssh.DefaultConfigPath, "path to the ssh config file.")
	return cmd
}

func shell(configpath *string) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		client, err := newClient(ctx, true)
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
				clog.Tipf("use \"coder workspaces rebuild %s\" to rebuild this workspace", workspace.Name),
			)
		}

		config, err := ssh.ReadConfig(ctx, *configpath)
		if err != nil {
			return xerrors.Errorf("read ssh config: %w", err)
		}

		if !config.ContainsHost(ssh.CoderHost(workspace.Name)) {
			clog.LogInfo(fmt.Sprintf("Host config not found for %q, writing config to %q\n", workspace.Name, *configpath))

			err := addWorkspaceConfig(ctx, client, config, false, *workspace)
			if err != nil {
				return xerrors.Errorf("add workspace config: %w", err)
			}

			err = config.Write()
			if err != nil {
				return xerrors.Errorf("write to config: %w", err)
			}
		}

		err = writeSSHKey(ctx, client)
		if err != nil {
			return err
		}
		ssh := exec.CommandContext(ctx,
			"ssh", fmt.Sprintf("coder.%s", workspace.Name),
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
		}
		return err
	}
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
				clog.Tipf("run \"coder workspaces ls\" to view your workspaces"),
			)
		}
		return clog.Error("missing [workspace_name] argument")
	}
	return nil
}
