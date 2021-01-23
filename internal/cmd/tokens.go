package cmd

import (
	"fmt"

	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/coder-cli/internal/x/xcobra"
	"cdr.dev/coder-cli/pkg/tablewriter"
	"github.com/spf13/cobra"
)

func tokensCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "tokens",
		Short:  "manage Coder API tokens for the active user",
		Hidden: true,
		Long: "Create and manage API Tokens for authenticating the CLI.\n" +
			"Statically authenticate using the token value with the " + "`" + "CODER_TOKEN" + "`" + " and " + "`" + "CODER_URL" + "`" + " environment variables.",
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
	return &cobra.Command{
		Use:   "ls",
		Short: "show the user's active API tokens",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := newClient(ctx)
			if err != nil {
				return err
			}

			tokens, err := client.APITokens(ctx, coder.Me)
			if err != nil {
				return err
			}

			err = tablewriter.WriteTable(len(tokens), func(i int) interface{} { return tokens[i] })
			if err != nil {
				return err
			}

			return nil
		},
	}
}

func createTokensCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create [token_name]",
		Short: "create generates a new API token and prints it to stdout",
		Args:  xcobra.ExactArgs(1, "token_name"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := newClient(ctx)
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
		Args:  xcobra.ExactArgs(1, "token_id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := newClient(ctx)
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
		Args:  xcobra.ExactArgs(1, "token_id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := newClient(ctx)
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
