package cmd

import (
	"cdr.dev/coder-cli/internal/coderutil"
	"cdr.dev/coder-cli/internal/x/xcobra"
	"cdr.dev/coder-cli/internal/x/xkube"
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
		installProviderCmd(),
		listProviderCmd(),
		deleteProviderCmd(),
		cordonProviderCmd(),
		unCordonProviderCmd(),
		renameProviderCmd(),
	)
	return cmd
}

func createProviderCmd() *cobra.Command {
	var (
		fromContext bool
		clusterCA string
		clusterAddr string
		SAToken string
		namespace string
		storageClass string
		sshEnabled bool
		environmentSA string
		orgAllowlist []string
	)
	cmd := &cobra.Command{
		Use:   "create [name]",
		Args:  xcobra.ExactArgs(1),
		Short: "create a new workspace provider.",
		Long:  "Create a new Coder workspace provider.",
		Example: `# create a new workspace provider

coder providers create my-provider`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var (
				ctx = cmd.Context()
				name = args[0]
				err error
				req = &coder.WorkspaceProviderKubernetesCreateRequest{
					Name: name,
					ClusterCA: clusterCA,
					ClusterAddress: clusterAddr,
					SAToken: SAToken,
					DefaultNamespace: namespace,
					StorageClass: storageClass,
					SSHEnabled: sshEnabled,
					EnvironmentSA: environmentSA,
					OrgAllowlist: orgAllowlist,
				}
			)

			if fromContext {
				req, err = xkube.ApplyKubeConfigFromContext(ctx, req)
				if err != nil {
					return xerrors.Errorf("reading kube config from context: %w", err)
				}
			}

			client, err := newClient(ctx, true)
			if err != nil {
				return err
			}

			_, err = client.CreateWorkspaceProvider(ctx, *req)
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&fromContext, "from-context", false, "use current kube context to retrieve cluster-cert, cluster-address, sa-token, and namespace if not set explicitly")
	cmd.Flags().StringVar(&clusterCA, "cluster-cert", "", "kubernetes cluster certificate")
	cmd.Flags().StringVar(&clusterAddr, "cluster-address", "", "kubernetes cluster address")
	cmd.Flags().StringVar(&SAToken, "sa-token", "", "kubernetes service account token")
	cmd.Flags().StringVar(&namespace, "namespace", "", "kubernetes namespace")
	cmd.Flags().StringVar(&storageClass, "storage-class", "", "storage class name to use for workspaces")
	cmd.Flags().BoolVar(&sshEnabled, "ssh-enabled", false, "enable ssh")
	cmd.Flags().StringVar(&environmentSA, "workspace-sa", "coder-", "kubernetes service account name to use for workspaces")
	cmd.Flags().StringArrayVar(&orgAllowlist, "org-allowlist", nil, "list organization IDs to allowlist")
	return cmd
}

func installProviderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install kubernetes resources for a new workspace provider.",
		Long:  "Install kubernetes resources for a new workspace provider.",
		Example: `# create a new workspace provider

coder providers install`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var (
				ctx = cmd.Context()
				err error
			)

			err = xkube.CreateWorkspaceProviderResources(ctx)
			if err != nil {
				return xerrors.Errorf("creating kubernetes resources: %w", err)
			}

			return nil
		},
	}

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

			client, err := newClient(ctx, true)
			if err != nil {
				return err
			}

			wps, err := client.WorkspaceProviders(ctx)
			if err != nil {
				return xerrors.Errorf("list workspace providers: %w", err)
			}

			err = tablewriter.WriteTable(cmd.OutOrStdout(), len(wps.Kubernetes), func(i int) interface{} {
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
			client, err := newClient(ctx, true)
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

func cordonProviderCmd() *cobra.Command {
	var reason string

	cmd := &cobra.Command{
		Use:   "cordon [workspace_provider_name]",
		Args:  xcobra.ExactArgs(1),
		Short: "cordon a workspace provider.",
		Long:  "Prevent an existing Coder workspace provider from supporting any additional workspaces.",
		Example: `# cordon an existing workspace provider by name
coder providers cordon my-workspace-provider --reason "limit cloud clost"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := newClient(ctx, true)
			if err != nil {
				return err
			}

			wpName := args[0]
			provider, err := coderutil.ProviderByName(ctx, client, wpName)
			if err != nil {
				return err
			}

			if err := client.CordonWorkspaceProvider(ctx, provider.ID, reason); err != nil {
				return err
			}
			clog.LogSuccess(fmt.Sprintf("provider %q successfully cordoned - you can no longer create workspaces on this provider without uncordoning first", wpName))
			return nil
		},
	}
	cmd.Flags().StringVar(&reason, "reason", "", "reason for cordoning the provider")
	_ = cmd.MarkFlagRequired("reason")
	return cmd
}

func unCordonProviderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uncordon [workspace_provider_name]",
		Args:  xcobra.ExactArgs(1),
		Short: "uncordon a workspace provider.",
		Long:  "Set a currently cordoned provider as ready; enabling it to continue provisioning resources for new workspaces.",
		Example: `# uncordon an existing workspace provider by name
coder providers uncordon my-workspace-provider`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := newClient(ctx, true)
			if err != nil {
				return err
			}

			wpName := args[0]
			provider, err := coderutil.ProviderByName(ctx, client, wpName)
			if err != nil {
				return err
			}

			if err := client.UnCordonWorkspaceProvider(ctx, provider.ID); err != nil {
				return err
			}
			clog.LogSuccess(fmt.Sprintf("provider %q successfully uncordoned - you can now create workspaces on this provider", wpName))
			return nil
		},
	}
	return cmd
}

func renameProviderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rename [old_name] [new_name]",
		Args:  xcobra.ExactArgs(2),
		Short: "rename a workspace provider.",
		Long:  "Changes the name field of an existing workspace provider.",
		Example: `# rename a workspace provider from 'built-in' to 'us-east-1'
coder providers rename build-in us-east-1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := newClient(ctx, true)
			if err != nil {
				return err
			}

			oldName := args[0]
			newName := args[1]
			provider, err := coderutil.ProviderByName(ctx, client, oldName)
			if err != nil {
				return err
			}

			if err := client.RenameWorkspaceProvider(ctx, provider.ID, newName); err != nil {
				return err
			}
			clog.LogSuccess(fmt.Sprintf("provider %s successfully renamed to %s", oldName, newName))
			return nil
		},
	}
	return cmd
}
