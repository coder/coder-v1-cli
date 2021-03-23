package cmd

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"golang.org/x/xerrors"

	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/coder-cli/internal/coderutil"
	"cdr.dev/coder-cli/internal/x/xcobra"
	"cdr.dev/coder-cli/pkg/clog"
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
	provider        string
	showEmptyGroups bool
}

func resourceTop() *cobra.Command {
	var options resourceTopOptions

	cmd := &cobra.Command{
		Use:   "top",
		Short: "resource viewer with Coder platform annotations",
		RunE:  runResourceTop(&options),
		Args:  xcobra.ExactArgs(0),
		Example: `coder resources top --group org
coder resources top --group org --verbose --org DevOps
coder resources top --group user --verbose --user name@example.com
coder resources top --group provider --verbose --provider myprovider
coder resources top --sort-by memory --show-empty`,
	}
	cmd.Flags().StringVar(&options.group, "group", "user", "the grouping parameter (user|org|provider)")
	cmd.Flags().StringVar(&options.user, "user", "", "filter by a user email")
	cmd.Flags().StringVar(&options.org, "org", "", "filter by the name of an organization")
	cmd.Flags().StringVar(&options.provider, "provider", "", "filter by the name of a workspace provider")
	cmd.Flags().StringVar(&options.sortBy, "sort-by", "cpu", "field to sort aggregate groups and environments by (cpu|memory)")
	cmd.Flags().BoolVar(&options.showEmptyGroups, "show-empty", false, "show groups with zero active environments")

	return cmd
}

func runResourceTop(options *resourceTopOptions) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		client, err := newClient(ctx)
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
		images, err := coderutil.MakeImageMap(ctx, client, envs)
		if err != nil {
			return xerrors.Errorf("get images: %w", err)
		}

		orgs, err := client.Organizations(ctx)
		if err != nil {
			return xerrors.Errorf("get organizations: %w", err)
		}

		providers, err := client.WorkspaceProviders(ctx)
		if err != nil {
			return xerrors.Errorf("get workspace providers: %w", err)
		}

		var groups []groupable
		var labeler envLabeler
		switch options.group {
		case "user":
			groups, labeler = aggregateByUser(providers.Kubernetes, users, orgs, envs, images, *options)
		case "org":
			groups, labeler = aggregateByOrg(providers.Kubernetes, users, orgs, envs, images, *options)
		case "provider":
			groups, labeler = aggregateByProvider(providers.Kubernetes, users, orgs, envs, images, *options)
		default:
			return xerrors.Errorf("unknown --group %q", options.group)
		}

		return printResourceTop(cmd.OutOrStdout(), groups, labeler, options.showEmptyGroups, options.sortBy)
	}
}

func aggregateByUser(providers []coder.KubernetesProvider, users []coder.User, orgs []coder.Organization, envs []coder.Environment, images map[string]*coder.Image, options resourceTopOptions) ([]groupable, envLabeler) {
	var groups []groupable
	providerIDMap := providerIDs(providers)
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
	return groups, labelAll(imgLabeler(images), providerLabeler(providerIDMap), orgLabeler(orgIDMap))
}

func userIDs(users []coder.User) map[string]coder.User {
	userIDMap := make(map[string]coder.User)
	for _, u := range users {
		userIDMap[u.ID] = u
	}
	return userIDMap
}

func aggregateByOrg(providers []coder.KubernetesProvider, users []coder.User, orgs []coder.Organization, envs []coder.Environment, images map[string]*coder.Image, options resourceTopOptions) ([]groupable, envLabeler) {
	var groups []groupable
	providerIDMap := providerIDs(providers)
	orgEnvs := make(map[string][]coder.Environment, len(orgs))
	userIDMap := userIDs(users)
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
	return groups, labelAll(userLabeler(userIDMap), imgLabeler(images), providerLabeler(providerIDMap))
}

func providerIDs(providers []coder.KubernetesProvider) map[string]coder.KubernetesProvider {
	providerIDMap := make(map[string]coder.KubernetesProvider)
	for _, p := range providers {
		providerIDMap[p.ID] = p
	}
	return providerIDMap
}

func aggregateByProvider(providers []coder.KubernetesProvider, users []coder.User, _ []coder.Organization, envs []coder.Environment, images map[string]*coder.Image, options resourceTopOptions) ([]groupable, envLabeler) {
	var groups []groupable
	providerIDMap := providerIDs(providers)
	userIDMap := userIDs(users)
	providerEnvs := make(map[string][]coder.Environment, len(providers))
	for _, e := range envs {
		if options.provider != "" && providerIDMap[e.ResourcePoolID].Name != options.provider {
			continue
		}
		providerEnvs[e.ResourcePoolID] = append(providerEnvs[e.ResourcePoolID], e)
	}
	for _, p := range providers {
		if options.provider != "" && p.Name != options.provider {
			continue
		}
		groups = append(groups, providerGrouping{provider: p, envs: providerEnvs[p.ID]})
	}
	return groups, labelAll(userLabeler(userIDMap), imgLabeler(images)) // TODO: consider adding an org label here
}

// groupable specifies a structure capable of being an aggregation group of environments (user, org, all).
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

type providerGrouping struct {
	provider coder.KubernetesProvider
	envs     []coder.Environment
}

func (p providerGrouping) environments() []coder.Environment {
	return p.envs
}

func (p providerGrouping) header() string {
	return fmt.Sprintf("%s\t", truncate(p.provider.Name, 20, "..."))
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
		clog.LogInfo(
			"no groups for the given filters exist with active environments",
			clog.Tipf("run \"--show-empty\" to see groups with no resources."),
		)
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

func labelAll(labels ...envLabeler) envLabeler { return multiLabeler(labels) }

type multiLabeler []envLabeler

func (m multiLabeler) label(e coder.Environment) string {
	var str strings.Builder
	for i, labeler := range m {
		if i != 0 {
			str.WriteString("\t")
		}
		str.WriteString(labeler.label(e))
	}
	return str.String()
}

type orgLabeler map[string]coder.Organization

func (o orgLabeler) label(e coder.Environment) string {
	return fmt.Sprintf("[org: %s]", o[e.OrganizationID].Name)
}

type imgLabeler map[string]*coder.Image

func (i imgLabeler) label(e coder.Environment) string {
	return fmt.Sprintf("[img: %s:%s]", i[e.ImageID].Repository, e.ImageTag)
}

type userLabeler map[string]coder.User

func (u userLabeler) label(e coder.Environment) string {
	return fmt.Sprintf("[user: %s]", u[e.UserID].Email)
}

type providerLabeler map[string]coder.KubernetesProvider

func (p providerLabeler) label(e coder.Environment) string {
	return fmt.Sprintf("[provider: %s]", p[e.ResourcePoolID].Name)
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
	cpuAllocation float32
	memAllocation float32

	// TODO: consider using these
	cpuUtilization float32
	memUtilization float32
}

func (a resources) String() string {
	return fmt.Sprintf(
		"[cpu: %.1fvCPU]\t[mem: %.1fGB]",
		a.cpuAllocation, a.memAllocation,
	)
}

// truncate the given string and replace the removed chars with some replacement (ex: "...").
func truncate(str string, max int, replace string) string {
	if len(str) <= max {
		return str
	}
	return str[:max+1] + replace
}
