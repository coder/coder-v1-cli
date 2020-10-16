package cmd

import (
	"encoding/json"
	"os"

	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/coder-cli/internal/x/xtabwriter"
	"github.com/spf13/cobra"
	"golang.org/x/xerrors"

	"go.coder.com/flog"
)

func makeEnvsCommand() *cobra.Command {
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
				flog.Info("no environments found")
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

	return cmd
}

func stopEnvCommand(user *string) *cobra.Command {
	return &cobra.Command{
		Use:   "stop [environment_name]",
		Short: "stop a Coder environment by name",
		Long:  "Stop a Coder environment by name",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return xerrors.Errorf("new client: %w", err)
			}

			envName := args[0]
			env, err := findEnv(cmd.Context(), client, envName, *user)
			if err != nil {
				return xerrors.Errorf("find environment by name: %w", err)
			}

			if err = client.StopEnvironment(cmd.Context(), env.ID); err != nil {
				return xerrors.Errorf("stop environment: %w", err)
			}
			flog.Success("Successfully stopped environment %q", envName)
			return nil
		},
	}
}
