package cmd

import (
	"fmt"

	"cdr.dev/coder-cli/coder-sdk"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"golang.org/x/xerrors"
)

func openCmd() *cobra.Command {
	var (
		workspaceName string
		user          string
	)

	cmd := &cobra.Command{
		Use:     "open",
		Short:   "Open Coder workspace IDE",
		Long:    "Open Coder workspace IDE in your default browser",
		Example: "coder open --workspace dev",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			client, err := newClient(ctx, true)
			if err != nil {
				return err
			}

			if workspaceName == "" {
				return xerrors.Errorf("Missing required parameter: --workspace")
			}

			workspaces, err := getWorkspaces(ctx, client, user)
			if err != nil {
				return err
			}

			workspaceID := ""
			for _, w := range workspaces {
				if w.Name == workspaceName {
					workspaceID = w.ID
				}
			}
			if workspaceID == "" {
				return xerrors.Errorf("Workspace %q not found", workspaceName)
			}

			u := client.BaseURL()
			u.Path += "/app/code"
			q := u.Query()
			q.Set("workspaceId", workspaceID)
			u.RawQuery = q.Encode()

			if err := browser.OpenURL(u.String()); err != nil {
				fmt.Printf("Open the following in your browser:\n\n\t%s\n\n", u.String())
			} else {
				fmt.Printf("Your browser has been opened to visit:\n\n\t%s\n\n", u.String())
			}

			return nil
		},
	}
	cmd.Flags().StringVar(&user, "user", coder.Me, "Specify the user whose resources to target")
	cmd.Flags().StringVar(&workspaceName, "workspace", "", "Name of workspace to open")

	return cmd
}
