package cmd

import (
	"fmt"
	"net/url"

	"cdr.dev/coder-cli/internal/coderutil"
	"cdr.dev/coder-cli/internal/x/xcobra"

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
		cordonProviderCmd(),
		unCordonProviderCmd(),
		renameProviderCmd(),
	)
	return cmd
}

func createProviderCmd() *cobra.Command {
	var (
		hostname       string
		clusterAddress string
	)
	cmd := &cobra.Command{
		Use:   "create [name] --hostname=[hostname] --cluster-address=[clusterAddress]",
		Args:  xcobra.ExactArgs(1),
		Short: "create a new workspace provider.",
		Long:  "Create a new Coder workspace provider.",
		Example: `# create a new workspace provider in a pending state

coder providers create my-provider --hostname=https://provider.example.com --cluster-address=https://255.255.255.255`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			client, err := newClient(ctx)
			if err != nil {
				return err
			}

			version, err := client.APIVersion(ctx)
			if err != nil {
				return xerrors.Errorf("get application version: %w", err)
			}

			cemanagerURL := client.BaseURL()
			ingressHost, err := url.Parse(hostname)
			if err != nil {
				return xerrors.Errorf("parse hostname: %w", err)
			}

			if cemanagerURL.Scheme != ingressHost.Scheme {
				return xerrors.Errorf("Coder access url and hostname must have matching protocols: coder access url: %s, workspace provider hostname: %s", cemanagerURL.String(), ingressHost.String())
			}

			// ExactArgs(1) ensures our name value can't panic on an out of bounds.
			createReq := &coder.CreateWorkspaceProviderReq{
				Name:           args[0],
				Type:           coder.WorkspaceProviderKubernetes,
				Hostname:       hostname,
				ClusterAddress: clusterAddress,
			}

			wp, err := client.CreateWorkspaceProvider(ctx, *createReq)
			if err != nil {
				return xerrors.Errorf("create workspace provider: %w", err)
			}

			var sslNote string
			if ingressHost.Scheme == "https" {
				sslNote = `
NOTE: Since the hostname provided is using https you must ensure the deployment
has a valid SSL certificate. See https://coder.com/docs/guides/ssl-certificates
for more information.`
			}

			clog.LogSuccess(fmt.Sprintf(`
Created workspace provider "%s"
`, createReq.Name))
			_ = tablewriter.WriteTable(cmd.OutOrStdout(), 1, func(i int) interface{} {
				return *wp
			})
			_, _ = fmt.Fprint(cmd.OutOrStdout(), `
Now that the workspace provider is provisioned, it must be deployed into the cluster. To learn more,
visit https://coder.com/docs/workspace-providers/deployment

When connected to the cluster you wish to deploy onto, use the following helm command:

helm upgrade coder-workspace-provider coder/workspace-provider \
    --version=`+version+` \
    --atomic \
    --install \
    --force \
    --set envproxy.token=`+wp.EnvproxyToken+` \
    --set envproxy.accessURL=`+ingressHost.String()+` \
    --set ingress.host=`+ingressHost.Hostname()+` \
    --set envproxy.clusterAddress=`+clusterAddress+` \
    --set cemanager.accessURL=`+cemanagerURL.String()+`
`+sslNote+`

WARNING: The 'envproxy.token' is a secret value that authenticates the workspace provider, 
make sure not to share this token or make it public. 

Other values can be set on the helm chart to further customize the deployment, see 
https://github.com/cdr/enterprise-helm/blob/workspace-providers-envproxy-only/README.md
`)

			return nil
		},
	}

	cmd.Flags().StringVar(&hostname, "hostname", "", "workspace provider hostname")
	cmd.Flags().StringVar(&clusterAddress, "cluster-address", "", "kubernetes cluster apiserver endpoint")
	_ = cmd.MarkFlagRequired("hostname")
	_ = cmd.MarkFlagRequired("cluster-address")
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
			client, err := newClient(ctx)
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
			client, err := newClient(ctx)
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
			client, err := newClient(ctx)
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
