package cmd

import (
	"encoding/json"
	"os"

	"cdr.dev/coder-cli/internal/x/tablewriter"
	"github.com/spf13/cobra"
	"golang.org/x/xerrors"
)

func usersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "users",
		Short: "Interact with Coder user accounts",
	}

	var outputFmt string
	lsCmd := &cobra.Command{
		Use:   "ls",
		Short: "list all user accounts",
		Example: `coder users ls -o json
coder users ls -o json | jq .[] | jq -r .email`,
		RunE: listUsers(&outputFmt),
	}
	lsCmd.Flags().StringVarP(&outputFmt, "output", "o", "human", "human | json")

	cmd.AddCommand(lsCmd)
	return cmd
}

func listUsers(outputFmt *string) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		client, err := newClient(ctx)
		if err != nil {
			return err
		}

		users, err := client.Users(ctx)
		if err != nil {
			return xerrors.Errorf("get users: %w", err)
		}

		switch *outputFmt {
		case "human":
			// For each element, return the user.
			each := func(i int) interface{} { return users[i] }
			if err := tablewriter.WriteTable(len(users), each); err != nil {
				return xerrors.Errorf("write table: %w", err)
			}
		case "json":
			if err := json.NewEncoder(os.Stdout).Encode(users); err != nil {
				return xerrors.Errorf("encode users as json: %w", err)
			}
		default:
			return xerrors.New("unknown value for --output")
		}
		return nil
	}
}
