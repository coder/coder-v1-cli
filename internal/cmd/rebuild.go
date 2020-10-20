package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cdr.dev/coder-cli/coder-sdk"
	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"go.coder.com/flog"
	"golang.org/x/xerrors"
)

func rebuildEnvCommand() *cobra.Command {
	var follow bool
	var force bool
	cmd := &cobra.Command{
		Use:   "rebuild [environment_name]",
		Short: "rebuild a Coder environment",
		Args:  cobra.ExactArgs(1),
		Example: `coder envs rebuild front-end-env --follow
coder envs rebuild backend-env --force`,
		Hidden: true, // TODO(@cmoog) un-hide
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := newClient()
			if err != nil {
				return err
			}
			env, err := findEnv(ctx, client, args[0], coder.Me)
			if err != nil {
				return err
			}

			if !force && env.LatestStat.ContainerStatus == coder.EnvironmentOn {
				_, err = (&promptui.Prompt{
					Label:     fmt.Sprintf("Rebuild environment \"%s\"? (will destroy any work outside of /home)", env.Name),
					IsConfirm: true,
				}).Run()
				if err != nil {
					return err
				}
			}

			if err = client.RebuildEnvironment(ctx, env.ID); err != nil {
				return err
			}
			if follow {
				if err = trailBuildLogs(ctx, client, env.ID); err != nil {
					return err
				}
			} else {
				flog.Info("Use \"coder envs watch-build %s\" to follow the build logs", env.Name)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&follow, "follow", false, "follow buildlog after initiating rebuild")
	cmd.Flags().BoolVar(&force, "force", false, "force rebuild without showing a confirmation prompt")
	return cmd
}

// trailBuildLogs follows the build log for a given environment and prints the staged
// output with loaders and success/failure indicators for each stage
func trailBuildLogs(ctx context.Context, client *coder.Client, envID string) error {
	const check = "✅"
	const failure = "❌"
	const loading = "⌛"

	newSpinner := func() *spinner.Spinner { return spinner.New(spinner.CharSets[11], 100*time.Millisecond) }

	logs, err := client.FollowEnvironmentBuildLog(ctx, envID)
	if err != nil {
		return err
	}
	var s *spinner.Spinner
	for l := range logs {
		if l.Err != nil {
			return l.Err
		}
		switch l.BuildLog.Type {
		case coder.BuildLogTypeStart:
			// the FE uses this to reset the UI
			// the CLI doesn't need to do anything here given that we only append to the trail
		case coder.BuildLogTypeStage:
			if s != nil {
				s.Stop()
				fmt.Print("\n")
			}
			s = newSpinner()
			msg := fmt.Sprintf("%s %s", l.BuildLog.Time.Format(time.RFC3339), l.BuildLog.Msg)
			s.Suffix = fmt.Sprintf("  -- %s", msg)
			s.FinalMSG = fmt.Sprintf("%s -- %s", check, msg)
			s.Start()
		case coder.BuildLogTypeSubstage:
			// TODO(@cmoog) add verbose substage printing
		case coder.BuildLogTypeError:
			if s != nil {
				s.FinalMSG = fmt.Sprintf("%s %s", failure, strings.TrimPrefix(s.Suffix, "  "))
				s.Stop()
			}
			fmt.Print(color.RedString("\t%s", l.BuildLog.Msg))
			s = newSpinner()
		case coder.BuildLogTypeDone:
			if s != nil {
				s.Stop()
			}
			return nil
		default:
			return xerrors.Errorf("unknown buildlog type: %s", l.BuildLog.Type)
		}
	}
	return nil
}

func watchBuildLogCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "watch-build [environment_name]",
		Example: "coder watch-build front-end-env",
		Short:   "trail the build log of a Coder environment",
		Args:    cobra.ExactArgs(1),
		Hidden:  true, // TODO(@cmoog) un-hide
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := newClient()
			if err != nil {
				return err
			}
			env, err := findEnv(ctx, client, args[0], coder.Me)
			if err != nil {
				return err
			}

			if err = trailBuildLogs(ctx, client, env.ID); err != nil {
				return err
			}
			return nil
		},
	}
	return cmd
}
