package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"golang.org/x/xerrors"

	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/coder-cli/pkg/clog"
	"cdr.dev/coder-cli/pkg/tablewriter"
)

func providersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "providers",
		Short:  "Interact with Coder workspace providers",
		Long:   "Perform operations on the Coder Workspace Providers for the platform.",
		Hidden: true,
	}

	cmd.AddCommand(
		createProviderCmd(),
		listProviderCmd(),
		deleteProviderCmd(),
	)
	return cmd
}

func createProviderCmd() *cobra.Command {
	var (
		name           string
		hostname       string
		clusterAddress string
	)
	cmd := &cobra.Command{
		Use:   "create --name=[name] --hostname=[hostname] --clusterAddress=[clusterAddress]",
		Short: "create a new workspace provider.",
		Long:  "Create a new Coder workspace provider.",
		Example: `# create a new workspace provider in a pending state

coder providers create --name=my-provider --hostname=provider.example.com --clusterAddress=255.255.255.255`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			client, err := newClient(ctx)
			if err != nil {
				return err
			}

			createReq := &coder.CreateWorkspaceProviderReq{
				Name:           name,
				Type:           coder.WorkspaceProviderKubernetes,
				Hostname:       hostname,
				ClusterAddress: clusterAddress,
			}

			wp, err := client.CreateWorkspaceProvider(ctx, *createReq)
			if err != nil {
				return xerrors.Errorf("create workspace provider: %w", err)
			}

			err = tablewriter.WriteTable(1, func(i int) interface{} {
				return *wp
			})
			if err != nil {
				return xerrors.Errorf("write table: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "workspace provider name")
	cmd.Flags().StringVar(&hostname, "hostname", "", "workspace provider hostname")
	cmd.Flags().StringVar(&clusterAddress, "clusterAddress", "", "kubernetes cluster apiserver endpoint")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("hostname")
	_ = cmd.MarkFlagRequired("clusterAdress")
	return cmd
}

func listProviderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ls",
		Short: "list workspace providers.",
		Long:  "List all Coder workspace providers.",
		Example: `# list workspace providers
coder providers ls`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			client, err := newClient(ctx)
			if err != nil {
				return err
			}

			wps, err := client.WorkspaceProviders(ctx)
			if err != nil {
				return xerrors.Errorf("list workspace providers: %w", err)
			}

			err = tablewriter.WriteTable(len(wps.Kubernetes), func(i int) interface{} {
				return wps.Kubernetes[i]
			})
			if err != nil {
				return xerrors.Errorf("write table: %w", err)
			}
			return nil
		},
	}
	return cmd
}

func deleteProviderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rm [workspace_provider_name]",
		Short: "remove a workspace provider.",
		Long:  "Remove an existing Coder workspace provider by name.",
		Example: `# remove an existing workspace provider by name
coder providers rm my-workspace-provider`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := newClient(ctx)
			if err != nil {
				return err
			}

			wps, err := client.WorkspaceProviders(ctx)
			if err != nil {
				return xerrors.Errorf("listing workspace providers: %w", err)
			}

			egroup := clog.LoggedErrGroup()
			for _, wpName := range args {
				name := wpName
				egroup.Go(func() error {
					var id string
					for _, wp := range wps.Kubernetes {
						if wp.Name == name {
							id = wp.ID
						}
					}
					if id == "" {
						return clog.Error(
							fmt.Sprintf(`failed to remove workspace provider "%s"`, name),
							clog.Causef(`no workspace provider found by name "%s"`, name),
						)
					}

					err = client.DeleteWorkspaceProviderByID(ctx, id)
					if err != nil {
						return clog.Error(
							fmt.Sprintf(`failed to remove workspace provider "%s"`, name),
							clog.Causef(err.Error()),
						)
					}

					clog.LogSuccess(fmt.Sprintf(`removed workspace provider with name "%s"`, name))

					return nil
				})
			}
			return egroup.Wait()
		},
	}
	return cmd
}
