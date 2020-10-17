package cmd

import (
	"fmt"
	"io"
	"os"
	"sort"
	"text/tabwriter"

	"cdr.dev/coder-cli/coder-sdk"
	"github.com/spf13/cobra"
	"golang.org/x/xerrors"
)

func makeResourceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "resources",
		Short:  "manager Coder resources with platform-level context (users, organizations, environments)",
		Hidden: true,
	}
	cmd.AddCommand(resourceTop())
	return cmd
}

func resourceTop() *cobra.Command {
	cmd := &cobra.Command{
		Use: "top",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := newClient()
			if err != nil {
				return err
			}

			// NOTE: it's not worth parrallelizing these calls yet given that this specific endpoint
			// takes about 20x times longer than the other two
			envs, err := client.Environments(ctx)
			if err != nil {
				return xerrors.Errorf("get environments %w", err)
			}

			userEnvs := make(map[string][]coder.Environment)
			for _, e := range envs {
				userEnvs[e.UserID] = append(userEnvs[e.UserID], e)
			}

			users, err := client.Users(ctx)
			if err != nil {
				return xerrors.Errorf("get users: %w", err)
			}

			orgIDMap := make(map[string]coder.Organization)
			orglist, err := client.Organizations(ctx)
			if err != nil {
				return xerrors.Errorf("get organizations: %w", err)
			}
			for _, o := range orglist {
				orgIDMap[o.ID] = o
			}

			printResourceTop(os.Stdout, users, orgIDMap, userEnvs)
			return nil
		},
	}

	return cmd
}

func printResourceTop(writer io.Writer, users []coder.User, orgIDMap map[string]coder.Organization, userEnvs map[string][]coder.Environment) {
	tabwriter := tabwriter.NewWriter(writer, 0, 0, 4, ' ', 0)
	defer func() { _ = tabwriter.Flush() }()

	var userResources []aggregatedUser
	for _, u := range users {
		// truncate user names to ensure tabwriter doesn't push our entire table too far
		u.Name = truncate(u.Name, 20, "...")
		userResources = append(userResources, aggregatedUser{User: u, resources: aggregateEnvResources(userEnvs[u.ID])})
	}
	sort.Slice(userResources, func(i, j int) bool {
		return userResources[i].cpuAllocation > userResources[j].cpuAllocation
	})

	for _, u := range userResources {
		_, _ = fmt.Fprintf(tabwriter, "%s\t(%s)\t%s", u.Name, u.Email, u.resources)
		if verbose {
			if len(userEnvs[u.ID]) > 0 {
				_, _ = fmt.Fprintf(tabwriter, "\f")
			}
			for _, env := range userEnvs[u.ID] {
				_, _ = fmt.Fprintf(tabwriter, "\t")
				_, _ = fmt.Fprintln(tabwriter, fmtEnvResources(env, orgIDMap))
			}
		}
		_, _ = fmt.Fprint(tabwriter, "\n")
	}
}

type aggregatedUser struct {
	coder.User
	resources
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

// truncate the given string and replace the removed chars with some replacement (ex: "...")
func truncate(str string, max int, replace string) string {
	if len(str) <= max {
		return str
	}
	return str[:max+1] + replace
}
