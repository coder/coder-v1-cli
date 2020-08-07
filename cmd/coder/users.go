package main

import (
	"encoding/json"
	"os"

	"cdr.dev/coder-cli/internal/x/xtabwriter"
	"github.com/spf13/cobra"
	"golang.org/x/xerrors"
)

func makeUsersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "users",
		Short: "Interact with Coder user accounts",
	}

	var outputFmt string
	lsCmd := &cobra.Command{
		Use:   "ls",
		Short: "list all user accounts",
		RunE:  listUsers(&outputFmt),
	}
	lsCmd.Flags().StringVarP(&outputFmt, "output", "0", "human", "human | json")

	cmd.AddCommand(lsCmd)
	return cmd
}

func listUsers(outputFmt *string) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		entClient := requireAuth()

		users, err := entClient.Users()
		if err != nil {
			return xerrors.Errorf("get users: %w", err)
		}

		switch *outputFmt {
		case "human":
			err := xtabwriter.WriteTable(len(users), func(i int) interface{} {
				return users[i]
			})
			if err != nil {
				return xerrors.Errorf("write table: %w", err)
			}
		case "json":
			err = json.NewEncoder(os.Stdout).Encode(users)
			if err != nil {
				return xerrors.Errorf("encode users as json: %w", err)
			}
		default:
			return xerrors.New("unknown value for --output")
		}
		return nil
	}
}
