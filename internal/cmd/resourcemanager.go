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
	var group string
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
			allEnvs, err := client.Environments(ctx)
			if err != nil {
				return xerrors.Errorf("get environments %w", err)
			}
			// only include environments whose last status was "ON"
			envs := make([]coder.Environment, 0)
			for _, e := range allEnvs {
				if e.LatestStat.ContainerStatus == coder.EnvironmentOn {
					envs = append(envs, e)
				}
			}

			users, err := client.Users(ctx)
			if err != nil {
				return xerrors.Errorf("get users: %w", err)
			}

			orgs, err := client.Organizations(ctx)
			if err != nil {
				return xerrors.Errorf("get organizations: %w", err)
			}

			var groups []groupable
			var labeler envLabeler
			switch group {
			case "user":
				userEnvs := make(map[string][]coder.Environment, len(users))
				for _, e := range envs {
					userEnvs[e.UserID] = append(userEnvs[e.UserID], e)
				}
				for _, u := range users {
					groups = append(groups, userGrouping{user: u, envs: userEnvs[u.ID]})
				}
				orgIDMap := make(map[string]coder.Organization)
				for _, o := range orgs {
					orgIDMap[o.ID] = o
				}
				labeler = orgLabeler{orgIDMap}
			case "org":
				orgEnvs := make(map[string][]coder.Environment, len(orgs))
				for _, e := range envs {
					orgEnvs[e.OrganizationID] = append(orgEnvs[e.OrganizationID], e)
				}
				for _, o := range orgs {
					groups = append(groups, orgGrouping{org: o, envs: orgEnvs[o.ID]})
				}
				userIDMap := make(map[string]coder.User)
				for _, u := range users {
					userIDMap[u.ID] = u
				}
				labeler = userLabeler{userIDMap}
			default:
				return xerrors.Errorf("unknown --group %q", group)
			}

			printResourceTop(os.Stdout, groups, labeler)
			return nil
		},
	}
	cmd.Flags().StringVar(&group, "group", "user", "the grouping parameter (user|org)")

	return cmd
}

// groupable specifies a structure capable of being an aggregation group of environments (user, org, all)
type groupable interface {
	header() string
	environments() []coder.Environment
}

type userGrouping struct {
	user coder.User
	envs []coder.Environment
}

func (u userGrouping) environments() []coder.Environment {
	return u.envs
}

func (u userGrouping) header() string {
	return fmt.Sprintf("%s\t(%s)", truncate(u.user.Name, 20, "..."), u.user.Email)
}

type orgGrouping struct {
	org  coder.Organization
	envs []coder.Environment
}

func (o orgGrouping) environments() []coder.Environment {
	return o.envs
}

func (o orgGrouping) header() string {
	plural := "s"
	if len(o.org.Members) < 2 {
		plural = ""
	}
	return fmt.Sprintf("%s\t(%v member%s)", truncate(o.org.Name, 20, "..."), len(o.org.Members), plural)
}

func printResourceTop(writer io.Writer, groups []groupable, labeler envLabeler) {
	tabwriter := tabwriter.NewWriter(writer, 0, 0, 4, ' ', 0)
	defer func() { _ = tabwriter.Flush() }()

	var userResources []aggregatedResources
	for _, group := range groups {
		// truncate user names to ensure tabwriter doesn't push our entire table too far
		userResources = append(userResources, aggregatedResources{groupable: group, resources: aggregateEnvResources(group.environments())})
	}
	sort.Slice(userResources, func(i, j int) bool {
		return userResources[i].cpuAllocation > userResources[j].cpuAllocation
	})

	for _, u := range userResources {
		_, _ = fmt.Fprintf(tabwriter, "%s\t%s", u.header(), u.resources)
		if verbose {
			if len(u.environments()) > 0 {
				_, _ = fmt.Fprintf(tabwriter, "\f")
			}
			for _, env := range u.environments() {
				_, _ = fmt.Fprintf(tabwriter, "\t")
				_, _ = fmt.Fprintln(tabwriter, fmtEnvResources(env, labeler))
			}
		}
		_, _ = fmt.Fprint(tabwriter, "\n")
	}
}

type aggregatedResources struct {
	groupable
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

func fmtEnvResources(env coder.Environment, labeler envLabeler) string {
	return fmt.Sprintf("%s\t%s\t%s", env.Name, resourcesFromEnv(env), labeler.label(env))
}

type envLabeler interface {
	label(coder.Environment) string
}

type orgLabeler struct {
	orgMap map[string]coder.Organization
}

func (o orgLabeler) label(e coder.Environment) string {
	return fmt.Sprintf("[org: %s]", o.orgMap[e.OrganizationID].Name)
}

type userLabeler struct {
	userMap map[string]coder.User
}

func (u userLabeler) label(e coder.Environment) string {
	return fmt.Sprintf("[user: %s]", u.userMap[e.UserID].Email)
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
