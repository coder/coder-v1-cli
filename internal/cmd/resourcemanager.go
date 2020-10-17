package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"cdr.dev/coder-cli/coder-sdk"
	"github.com/spf13/cobra"
)

func makeResourceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resources",
		Short: "manager Coder resources with platform-level context (users, organizations, environments)",
	}
	cmd.AddCommand(resourceTop)
	return cmd
}

var resourceTop = &cobra.Command{
	Use: "top",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		client, err := newClient()
		if err != nil {
			return err
		}

		envs, err := client.ListEnvironments(ctx)
		if err != nil {
			return err
		}

		userEnvs := make(map[string][]coder.Environment)
		for _, e := range envs {
			userEnvs[e.UserID] = append(userEnvs[e.UserID], e)
		}

		users, err := client.Users(ctx)
		if err != nil {
			return err
		}

		orgs := make(map[string]coder.Organization)
		orglist, err := client.Organizations(ctx)
		if err != nil {
			return err
		}
		for _, o := range orglist {
			orgs[o.ID] = o
		}

		tabwriter := tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', 0)
		for _, u := range users {
			_, _ = fmt.Fprintf(tabwriter, "%s\t(%s)\t%s", u.Name, u.Email, aggregateEnvResources(userEnvs[u.ID]))
			if len(userEnvs[u.ID]) > 0 {
				_, _ = fmt.Fprintf(tabwriter, "\f")
			}
			for _, env := range userEnvs[u.ID] {
				_, _ = fmt.Fprintf(tabwriter, "\t")
				_, _ = fmt.Fprintln(tabwriter, fmtEnvResources(env, orgs))
			}
			fmt.Fprint(tabwriter, "\n")
		}
		_ = tabwriter.Flush()

		return nil
	},
}

func resourcesFromEnv(env coder.Environment) resources {
	return resources{
		cpuAllocation:  env.CPUCores,
		cpuUtilization: env.LatestStat.CPUUsage,
		memAllocation:  env.MemoryGB,
		memUtilization: env.LatestStat.MemoryUsage,
	}
}

func fmtEnvResources(env coder.Environment, orgs map[string]coder.Organization) string {
	return fmt.Sprintf("%s\t%s\t[org: %s]", env.Name, resourcesFromEnv(env), orgs[env.OrganizationID].Name)
}

func aggregateEnvResources(envs []coder.Environment) resources {
	var aggregate resources
	for _, e := range envs {
		aggregate.cpuAllocation += e.CPUCores
		aggregate.cpuUtilization += e.LatestStat.CPUUsage
		aggregate.memAllocation += e.MemoryGB
		aggregate.memUtilization += e.LatestStat.MemoryUsage
	}
	return aggregate
}

type resources struct {
	cpuAllocation  float32
	cpuUtilization float32
	memAllocation  float32
	memUtilization float32
}

func (a resources) String() string {
	return fmt.Sprintf("[cpu: alloc=%.1fvCPU, util=%.1f]\t[mem: alloc=%.1fGB, util=%.1f]", a.cpuAllocation, a.cpuUtilization, a.memAllocation, a.memUtilization)
}
