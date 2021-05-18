package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"golang.org/x/xerrors"

	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/coder-cli/internal/x/xcobra"
	"cdr.dev/coder-cli/pkg/tablewriter"
)

func tokensCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tokens",
		Short: "manage Coder API tokens for the active user",
		Long: "Create and manage API Tokens for authenticating the CLI.\n" +
			"Statically authenticate using the token value with the " + "`" + "CODER_TOKEN" + "`" + " and " + "`" + "CODER_URL" + "`" + " workspace variables.",
	}
	cmd.AddCommand(
		lsTokensCmd(),
		createTokensCmd(),
		rmTokenCmd(),
		regenTokenCmd(),
	)
	return cmd
}

func lsTokensCmd() *cobra.Command {
	var outputFmt string

	cmd := &cobra.Command{
		Use:   "ls",
		Short: "show the user's active API tokens",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := newClient(ctx, true)
			if err != nil {
				return err
			}

			tokens, err := client.APITokens(ctx, coder.Me)
			if err != nil {
				return err
			}

			switch outputFmt {
			case humanOutput:
				err := tablewriter.WriteTable(cmd.OutOrStdout(), len(tokens), func(i int) interface{} {
					return tokens[i]
				})
				if err != nil {
					return xerrors.Errorf("write table: %w", err)
				}
			case jsonOutput:
				err := json.NewEncoder(cmd.OutOrStdout()).Encode(tokens)
				if err != nil {
					return xerrors.Errorf("write tokens as JSON: %w", err)
				}
			default:
				return xerrors.Errorf("unknown --output value %q", outputFmt)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&outputFmt, "output", "o", humanOutput, "human | json")

	return cmd
}

func createTokensCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create [token_name]",
		Short: "create generates a new API token and prints it to stdout",
		Args:  xcobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := newClient(ctx, true)
			if err != nil {
				return err
			}
			token, err := client.CreateAPIToken(ctx, coder.Me, coder.CreateAPITokenReq{
				Name: args[0],
			})
			if err != nil {
				return err
			}
			fmt.Println(token)
			return nil
		},
	}
}

func rmTokenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm [token_id]",
		Short: "remove an API token by its unique ID",
		Args:  xcobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := newClient(ctx, true)
			if err != nil {
				return err
			}
			if err = client.DeleteAPIToken(ctx, coder.Me, args[0]); err != nil {
				return err
			}
			return nil
		},
	}
}

func regenTokenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "regen [token_id]",
		Short: "regenerate an API token by its unique ID and print the new token to stdout",
		Args:  xcobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := newClient(ctx, true)
			if err != nil {
				return err
			}
			token, err := client.RegenerateAPIToken(ctx, coder.Me, args[0])
			if err != nil {
				return nil
			}
			fmt.Println(token)
			return nil
		},
	}
}
