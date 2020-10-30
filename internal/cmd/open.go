package cmd

import (
	"context"
	"fmt"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/coder-cli/pkg/clog"
	"github.com/spf13/cobra"
)

func openCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "open [environment_name]:[remote_workspace_path]",
		Args:              cobra.ExactArgs(1),
		Short:             "launch your IDE and connect to a Coder environment",
		ValidArgsFunction: getEnvsForCompletion(coder.Me),
		Example: `coder open backend-dev:/home/coder
coder open backend-dev:/home/coder/backend-project`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := newClient(ctx)
			if err != nil {
				return err
			}
			parts := strings.Split(args[0], ":")
			if len(parts) < 2 {
				return cmd.Usage()
			}

			envName, workspacePath := parts[0], parts[1]
			env, err := findEnv(ctx, client, envName, coder.Me)
			if err != nil {
				return err
			}
			warnIfAliasMissing(env.Name)

			if _, err := exec.LookPath("code"); err != nil {
				return clog.Error(
					`"code" command line tool not found`, clog.BlankLine,
					clog.Tipf(`read about "code" here: https://code.visualstudio.com/docs/editor/command-line`),
				)
			}

			// TODO(@cmoog) maybe check if it's not installed first, although it does seem to check internally
			if err = installRemoteSSH(ctx); err != nil {
				return clog.Error(
					"failed to install VSCode Remote SSH extension", clog.BlankLine,
					clog.Causef(err.Error()),
				)
			}

			if err = openVSCodeRemote(ctx, env.Name, workspacePath); err != nil {
				return err
			}

			return nil
		},
	}
}

func warnIfAliasMissing(envName string) {
	usr, err := user.Current()
	if err != nil {
		return
	}

	// TODO(@cmoog) might be a better way of finding the path of the SSH config / whether an alias is valid or not
	config, err := readStr(filepath.Join(usr.HomeDir, ".ssh", "config"))
	if err != nil {
		clog.LogWarn(
			"failed to check that SSH target exists", clog.BlankLine,
			clog.Causef(err.Error()),
			clog.Tipf(`run "coder config-ssh" to add SSH targets for each environment`),
		)
		return
	}

	if !strings.Contains(config, fmt.Sprintf("coder.%s", envName)) {
		clog.LogWarn(
			"SSH alias not found for environment", clog.BlankLine,
			clog.Tipf(`run "coder config-ssh" to add SSH targets for each environment`),
		)
		return
	}
}

func installRemoteSSH(ctx context.Context) error {
	cmd := exec.CommandContext(ctx,
		"code", "--install-extension", "ms-vscode-remote.remote-ssh",
	)
	if _, err := cmd.CombinedOutput(); err != nil {
		return err
	}
	return nil
}

func openVSCodeRemote(ctx context.Context, envName, dirPath string) error {
	cmd := exec.CommandContext(ctx,
		"code", "--remote", "ssh-remote+coder."+envName, dirPath,
	)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}
	return nil
}
