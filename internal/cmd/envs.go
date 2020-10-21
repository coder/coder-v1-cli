package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sync/atomic"

	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/coder-cli/internal/clog"
	"cdr.dev/coder-cli/internal/x/xtabwriter"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"golang.org/x/xerrors"
)

func envsCommand() *cobra.Command {
	var outputFmt string
	var user string
	cmd := &cobra.Command{
		Use:   "envs",
		Short: "Interact with Coder environments",
		Long:  "Perform operations on the Coder environments owned by the active user.",
	}
	cmd.PersistentFlags().StringVar(&user, "user", coder.Me, "Specify the user whose resources to target")

	lsCmd := &cobra.Command{
		Use:   "ls",
		Short: "list all environments owned by the active user",
		Long:  "List all Coder environments owned by the active user.",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return err
			}
			envs, err := getEnvs(cmd.Context(), client, user)
			if err != nil {
				return err
			}
			if len(envs) < 1 {
				clog.LogInfo("no environments found")
				return nil
			}

			switch outputFmt {
			case "human":
				err := xtabwriter.WriteTable(len(envs), func(i int) interface{} {
					return envs[i]
				})
				if err != nil {
					return xerrors.Errorf("write table: %w", err)
				}
			case "json":
				err := json.NewEncoder(os.Stdout).Encode(envs)
				if err != nil {
					return xerrors.Errorf("write environments as JSON: %w", err)
				}
			default:
				return xerrors.Errorf("unknown --output value %q", outputFmt)
			}
			return nil
		},
	}
	lsCmd.Flags().StringVarP(&outputFmt, "output", "o", "human", "human | json")
	cmd.AddCommand(lsCmd)
	cmd.AddCommand(stopEnvCommand(&user))

	cmd.AddCommand(watchBuildLogCommand())
	cmd.AddCommand(rebuildEnvCommand())
	return cmd
}

func stopEnvCommand(user *string) *cobra.Command {
	return &cobra.Command{
		Use:   "stop [...environment_names]",
		Short: "stop Coder environments by name",
		Long:  "Stop Coder environments by name",
		Example: `coder envs stop front-end-env
coder envs stop front-end-env backend-env

# stop all of your environments
coder envs ls -o json | jq -c '.[].name' | xargs coder envs stop

# stop all environments for a given user
coder envs --user charlie@coder.com ls -o json \
	| jq -c '.[].name' \
	| xargs coder envs --user charlie@coder.com stop`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return xerrors.Errorf("new client: %w", err)
			}

			var egroup errgroup.Group
			var fails int32
			for _, envName := range args {
				envName := envName
				egroup.Go(func() error {
					env, err := findEnv(cmd.Context(), client, envName, *user)
					if err != nil {
						atomic.AddInt32(&fails, 1)
						clog.Log(err)
						return xerrors.Errorf("find env by name: %w", err)
					}

					if err = client.StopEnvironment(cmd.Context(), env.ID); err != nil {
						atomic.AddInt32(&fails, 1)
						err = clog.Fatal(fmt.Sprintf("stop environment %q", env.Name),
							clog.Cause(err.Error()), clog.BlankLine,
							clog.Hint("current environment status is %q", env.LatestStat.ContainerStatus),
						)
						clog.Log(err)
						return err
					}
					clog.LogSuccess(fmt.Sprintf("successfully stopped environment %q", envName))
					return nil
				})
			}

			if err = egroup.Wait(); err != nil {
				return clog.Fatal(fmt.Sprintf("%d failure(s) emitted", fails))
			}
			return nil
		},
	}
}
