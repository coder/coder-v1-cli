package cmd

import (
	"fmt"
	"net/url"

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
	)
	return cmd
}

func createProviderCmd() *cobra.Command {
	var (
		hostname       string
		clusterAddress string
	)
	cmd := &cobra.Command{
		Use:   "create [name] --hostname=[hostname] --clusterAddress=[clusterAddress]",
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

			cemanagerURL := client.BaseURL()
			ingressHost, err := url.Parse(hostname)
			if err != nil {
				return xerrors.Errorf("parse hostname: %w", err)
			}

			version, err := client.APIVersion(ctx)
			if err != nil {
				return xerrors.Errorf("get application version: %w", err)
			}

			clog.LogSuccess(fmt.Sprintf(`
Created workspace provider "%s"
`, createReq.Name))
			_ = tablewriter.WriteTable(cmd.OutOrStdout(), 1, func(i int) interface{} {
				return *wp
			})
			_, _ = fmt.Fprint(cmd.OutOrStdout(), `
Now that the workspace provider is provisioned, it must be deployed into the cluster. To learn more,
visit https://coder.com/docs/guides/deploying-workspace-provider

When connected to the cluster you wish to deploy onto, use the following helm command:

helm upgrade coder-workspace-provider coder/workspace-provider \
    --version=`+version+` \
    --atomic \
    --install \
    --force \
    --set envproxy.token=`+wp.EnvproxyToken+` \
    --set ingress.host=`+ingressHost.Hostname()+` \
    --set envproxy.clusterAddress=`+clusterAddress+` \
    --set cemanager.accessURL=`+cemanagerURL.String()+`

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
