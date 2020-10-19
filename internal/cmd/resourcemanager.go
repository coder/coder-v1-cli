package cmd

import (
	"fmt"
	"io"
	"os"
	"sort"
	"text/tabwriter"

	"cdr.dev/coder-cli/coder-sdk"
	"github.com/spf13/cobra"
	"go.coder.com/flog"
	"golang.org/x/xerrors"
)

func resourceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "resources",
		Short:  "manage Coder resources with platform-level context (users, organizations, environments)",
		Hidden: true,
	}
	cmd.AddCommand(resourceTop())
	return cmd
}

type resourceTopOptions struct {
	group           string
	user            string
	org             string
	sortBy          string
	showEmptyGroups bool
}

func resourceTop() *cobra.Command {
	var options resourceTopOptions

	cmd := &cobra.Command{
		Use:   "top",
		Short: "resource viewer with Coder platform annotations",
		RunE:  runResourceTop(&options),
		Example: `coder resources top --group org
coder resources top --group org --verbose --org DevOps
coder resources top --group user --verbose --user name@example.com
coder resources top --sort-by memory --show-empty`,
	}
	cmd.Flags().StringVar(&options.group, "group", "user", "the grouping parameter (user|org)")
	cmd.Flags().StringVar(&options.user, "user", "", "filter by a user email")
	cmd.Flags().StringVar(&options.org, "org", "", "filter by the name of an organization")
	cmd.Flags().StringVar(&options.sortBy, "sort-by", "cpu", "field to sort aggregate groups and environments by (cpu|memory)")
	cmd.Flags().BoolVar(&options.showEmptyGroups, "show-empty", false, "show groups with zero active environments")

	return cmd
}

func runResourceTop(options *resourceTopOptions) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
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
		switch options.group {
		case "user":
			groups, labeler = aggregateByUser(users, orgs, envs, *options)
		case "org":
			groups, labeler = aggregateByOrg(users, orgs, envs, *options)
		default:
			return xerrors.Errorf("unknown --group %q", options.group)
		}

		return printResourceTop(os.Stdout, groups, labeler, options.showEmptyGroups, options.sortBy)
	}
}

func aggregateByUser(users []coder.User, orgs []coder.Organization, envs []coder.Environment, options resourceTopOptions) ([]groupable, envLabeler) {
	var groups []groupable
	orgIDMap := make(map[string]coder.Organization)
	for _, o := range orgs {
		orgIDMap[o.ID] = o
	}
	userEnvs := make(map[string][]coder.Environment, len(users))
	for _, e := range envs {
		if options.org != "" && orgIDMap[e.OrganizationID].Name != options.org {
			continue
		}
		userEnvs[e.UserID] = append(userEnvs[e.UserID], e)
	}
	for _, u := range users {
		if options.user != "" && u.Email != options.user {
			continue
		}
		groups = append(groups, userGrouping{user: u, envs: userEnvs[u.ID]})
	}
	return groups, orgLabeler{orgIDMap}
}

func aggregateByOrg(users []coder.User, orgs []coder.Organization, envs []coder.Environment, options resourceTopOptions) ([]groupable, envLabeler) {
	var groups []groupable
	userIDMap := make(map[string]coder.User)
	for _, u := range users {
		userIDMap[u.ID] = u
	}
	orgEnvs := make(map[string][]coder.Environment, len(orgs))
	for _, e := range envs {
		if options.user != "" && userIDMap[e.UserID].Email != options.user {
			continue
		}
		orgEnvs[e.OrganizationID] = append(orgEnvs[e.OrganizationID], e)
	}
	for _, o := range orgs {
		if options.org != "" && o.Name != options.org {
			continue
		}
		groups = append(groups, orgGrouping{org: o, envs: orgEnvs[o.ID]})
	}
	return groups, userLabeler{userIDMap}
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

func printResourceTop(writer io.Writer, groups []groupable, labeler envLabeler, showEmptyGroups bool, sortBy string) error {
	tabwriter := tabwriter.NewWriter(writer, 0, 0, 4, ' ', 0)
	defer func() { _ = tabwriter.Flush() }()

	var userResources []aggregatedResources
	for _, group := range groups {
		if !showEmptyGroups && len(group.environments()) < 1 {
			continue
		}
		userResources = append(userResources, aggregatedResources{
			groupable: group, resources: aggregateEnvResources(group.environments()),
		})
	}

	err := sortAggregatedResources(userResources, sortBy)
	if err != nil {
		return err
	}

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
	if len(userResources) == 0 {
		flog.Info("No groups for the given filters exist with active environments.")
		flog.Info("Use \"--show-empty\" to see groups with no resources.")
	}
	return nil
}

func sortAggregatedResources(resources []aggregatedResources, sortBy string) error {
	const cpu = "cpu"
	const memory = "memory"
	switch sortBy {
	case cpu:
		sort.Slice(resources, func(i, j int) bool {
			return resources[i].cpuAllocation > resources[j].cpuAllocation
		})
	case memory:
		sort.Slice(resources, func(i, j int) bool {
			return resources[i].memAllocation > resources[j].memAllocation
		})
	default:
		return xerrors.Errorf("unknown --sort-by value of \"%s\"", sortBy)
	}
	for _, group := range resources {
		envs := group.environments()
		switch sortBy {
		case cpu:
			sort.Slice(envs, func(i, j int) bool { return envs[i].CPUCores > envs[j].CPUCores })
		case memory:
			sort.Slice(envs, func(i, j int) bool { return envs[i].MemoryGB > envs[j].MemoryGB })
		default:
			return xerrors.Errorf("unknown --sort-by value of \"%s\"", sortBy)
		}
	}
	return nil
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
	return fmt.Sprintf(
		"[cpu: alloc=%.1fvCPU]\t[mem: alloc=%.1fGB]",
		a.cpuAllocation, a.memAllocation,
	)

	// TODO@cmoog: consider adding the utilization info once a historical average is considered or implemented
	// return fmt.Sprintf(
	// 	"[cpu: alloc=%.1fvCPU, util=%s]\t[mem: alloc=%.1fGB, util=%s]",
	// 	a.cpuAllocation, a.cpuUtilPercentage(), a.memAllocation, a.memUtilPercentage(),
	// )
}

func (a resources) cpuUtilPercentage() string {
	if a.cpuAllocation == 0 {
		return "N/A"
	}
	return fmt.Sprintf("%.1f%%", a.cpuUtilization/a.cpuAllocation*100)
}

func (a resources) memUtilPercentage() string {
	if a.memAllocation == 0 {
		return "N/A"
	}
	return fmt.Sprintf("%.1f%%", a.memUtilization/a.memAllocation*100)
}

// truncate the given string and replace the removed chars with some replacement (ex: "...")
func truncate(str string, max int, replace string) string {
	if len(str) <= max {
		return str
	}
	return str[:max+1] + replace
}
