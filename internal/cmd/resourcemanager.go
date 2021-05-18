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
		Short:  "manage Coder resources with platform-level context (users, organizations, workspaces)",
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
	cmd.Flags().StringVar(&options.sortBy, "sort-by", "cpu", "field to sort aggregate groups and workspaces by (cpu|memory)")
	cmd.Flags().BoolVar(&options.showEmptyGroups, "show-empty", false, "show groups with zero active workspaces")

	return cmd
}

func runResourceTop(options *resourceTopOptions) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		client, err := newClient(ctx, true)
		if err != nil {
			return err
		}

		// NOTE: it's not worth parrallelizing these calls yet given that this specific endpoint
		// takes about 20x times longer than the other two
		allWorkspaces, err := client.Workspaces(ctx)
		if err != nil {
			return xerrors.Errorf("get workspaces %w", err)
		}
		// only include workspaces whose last status was "ON"
		workspaces := make([]coder.Workspace, 0)
		for _, e := range allWorkspaces {
			if e.LatestStat.ContainerStatus == coder.WorkspaceOn {
				workspaces = append(workspaces, e)
			}
		}

		users, err := client.Users(ctx)
		if err != nil {
			return xerrors.Errorf("get users: %w", err)
		}
		images, err := coderutil.MakeImageMap(ctx, client, workspaces)
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
		data := entities{
			providers:  providers.Kubernetes,
			users:      users,
			orgs:       orgs,
			workspaces: workspaces,
			images:     images,
		}
		return presentEntites(cmd.OutOrStdout(), data, *options)
	}
}

func presentEntites(w io.Writer, data entities, options resourceTopOptions) error {
	var (
		groups  []groupable
		labeler workspaceLabeler
	)
	switch options.group {
	case "user":
		groups, labeler = aggregateByUser(data, options)
	case "org":
		groups, labeler = aggregateByOrg(data, options)
	case "provider":
		groups, labeler = aggregateByProvider(data, options)
	default:
		return xerrors.Errorf("unknown --group %q", options.group)
	}

	return printResourceTop(w, groups, labeler, options.showEmptyGroups, options.sortBy)
}

type entities struct {
	providers  []coder.KubernetesProvider
	users      []coder.User
	orgs       []coder.Organization
	workspaces []coder.Workspace
	images     map[string]*coder.Image
}

func aggregateByUser(data entities, options resourceTopOptions) ([]groupable, workspaceLabeler) {
	var groups []groupable
	providerIDMap := providerIDs(data.providers)
	orgIDMap := make(map[string]coder.Organization)
	for _, o := range data.orgs {
		orgIDMap[o.ID] = o
	}
	userWorkspaces := make(map[string][]coder.Workspace, len(data.users))
	for _, e := range data.workspaces {
		if options.org != "" && orgIDMap[e.OrganizationID].Name != options.org {
			continue
		}
		userWorkspaces[e.UserID] = append(userWorkspaces[e.UserID], e)
	}
	for _, u := range data.users {
		if options.user != "" && u.Email != options.user {
			continue
		}
		groups = append(groups, userGrouping{user: u, userWorkspaces: userWorkspaces[u.ID]})
	}
	return groups, labelAll(imgLabeler(data.images), providerLabeler(providerIDMap), orgLabeler(orgIDMap))
}

func userIDs(users []coder.User) map[string]coder.User {
	userIDMap := make(map[string]coder.User)
	for _, u := range users {
		userIDMap[u.ID] = u
	}
	return userIDMap
}

func aggregateByOrg(data entities, options resourceTopOptions) ([]groupable, workspaceLabeler) {
	var groups []groupable
	providerIDMap := providerIDs(data.providers)
	orgWorkspaces := make(map[string][]coder.Workspace, len(data.orgs))
	userIDMap := userIDs(data.users)
	for _, e := range data.workspaces {
		if options.user != "" && userIDMap[e.UserID].Email != options.user {
			continue
		}
		orgWorkspaces[e.OrganizationID] = append(orgWorkspaces[e.OrganizationID], e)
	}
	for _, o := range data.orgs {
		if options.org != "" && o.Name != options.org {
			continue
		}
		groups = append(groups, orgGrouping{org: o, orgWorkspaces: orgWorkspaces[o.ID]})
	}
	return groups, labelAll(imgLabeler(data.images), userLabeler(userIDMap), providerLabeler(providerIDMap))
}

func providerIDs(providers []coder.KubernetesProvider) map[string]coder.KubernetesProvider {
	providerIDMap := make(map[string]coder.KubernetesProvider)
	for _, p := range providers {
		providerIDMap[p.ID] = p
	}
	return providerIDMap
}

func aggregateByProvider(data entities, options resourceTopOptions) ([]groupable, workspaceLabeler) {
	var groups []groupable
	providerIDMap := providerIDs(data.providers)
	userIDMap := userIDs(data.users)
	providerWorkspaces := make(map[string][]coder.Workspace, len(data.providers))
	for _, e := range data.workspaces {
		if options.provider != "" && providerIDMap[e.ResourcePoolID].Name != options.provider {
			continue
		}
		providerWorkspaces[e.ResourcePoolID] = append(providerWorkspaces[e.ResourcePoolID], e)
	}
	for _, p := range data.providers {
		if options.provider != "" && p.Name != options.provider {
			continue
		}
		groups = append(groups, providerGrouping{provider: p, providerWorkspaces: providerWorkspaces[p.ID]})
	}
	return groups, labelAll(imgLabeler(data.images), userLabeler(userIDMap)) // TODO: consider adding an org label here
}

// groupable specifies a structure capable of being an aggregation group of workspaces (user, org, all).
type groupable interface {
	header() string
	workspaces() []coder.Workspace
}

type userGrouping struct {
	user           coder.User
	userWorkspaces []coder.Workspace
}

func (u userGrouping) workspaces() []coder.Workspace {
	return u.userWorkspaces
}

func (u userGrouping) header() string {
	return fmt.Sprintf("%s\t(%s)", truncate(u.user.Name, 20, "..."), u.user.Email)
}

type orgGrouping struct {
	org           coder.Organization
	orgWorkspaces []coder.Workspace
}

func (o orgGrouping) workspaces() []coder.Workspace {
	return o.orgWorkspaces
}

func (o orgGrouping) header() string {
	plural := "s"
	if len(o.org.Members) == 1 {
		plural = ""
	}
	return fmt.Sprintf("%s\t(%v member%s)", truncate(o.org.Name, 20, "..."), len(o.org.Members), plural)
}

type providerGrouping struct {
	provider           coder.KubernetesProvider
	providerWorkspaces []coder.Workspace
}

func (p providerGrouping) workspaces() []coder.Workspace {
	return p.providerWorkspaces
}

func (p providerGrouping) header() string {
	return fmt.Sprintf("%s\t", truncate(p.provider.Name, 20, "..."))
}

func printResourceTop(writer io.Writer, groups []groupable, labeler workspaceLabeler, showEmptyGroups bool, sortBy string) error {
	tabwriter := tabwriter.NewWriter(writer, 0, 0, 4, ' ', 0)
	defer func() { _ = tabwriter.Flush() }()

	var userResources []aggregatedResources
	for _, group := range groups {
		if !showEmptyGroups && len(group.workspaces()) < 1 {
			continue
		}
		userResources = append(userResources, aggregatedResources{
			groupable: group, resources: aggregateWorkspaceResources(group.workspaces()),
		})
	}

	err := sortAggregatedResources(userResources, sortBy)
	if err != nil {
		return err
	}

	for _, u := range userResources {
		_, _ = fmt.Fprintf(tabwriter, "%s\t%s", u.header(), u.resources)
		if verbose {
			if len(u.workspaces()) > 0 {
				_, _ = fmt.Fprintf(tabwriter, "\f")
			}
			for _, workspace := range u.workspaces() {
				_, _ = fmt.Fprintf(tabwriter, "\t")
				_, _ = fmt.Fprintln(tabwriter, fmtWorkspaceResources(workspace, labeler))
			}
		}
		_, _ = fmt.Fprint(tabwriter, "\n")
	}
	if len(userResources) == 0 {
		clog.LogInfo(
			"no groups for the given filters exist with active workspaces",
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
		workspaces := group.workspaces()
		switch sortBy {
		case cpu:
			sort.Slice(workspaces, func(i, j int) bool { return workspaces[i].CPUCores > workspaces[j].CPUCores })
		case memory:
			sort.Slice(workspaces, func(i, j int) bool { return workspaces[i].MemoryGB > workspaces[j].MemoryGB })
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

func resourcesFromWorkspace(workspace coder.Workspace) resources {
	return resources{
		cpuAllocation:  workspace.CPUCores,
		cpuUtilization: workspace.LatestStat.CPUUsage,
		memAllocation:  workspace.MemoryGB,
		memUtilization: workspace.LatestStat.MemoryUsage,
	}
}

func fmtWorkspaceResources(workspace coder.Workspace, labeler workspaceLabeler) string {
	return fmt.Sprintf("%s\t%s\t%s", truncate(workspace.Name, 20, "..."), resourcesFromWorkspace(workspace), labeler.label(workspace))
}

type workspaceLabeler interface {
	label(coder.Workspace) string
}

func labelAll(labels ...workspaceLabeler) workspaceLabeler { return multiLabeler(labels) }

type multiLabeler []workspaceLabeler

func (m multiLabeler) label(e coder.Workspace) string {
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

func (o orgLabeler) label(e coder.Workspace) string {
	return fmt.Sprintf("[org: %s]", o[e.OrganizationID].Name)
}

type imgLabeler map[string]*coder.Image

func (i imgLabeler) label(e coder.Workspace) string {
	return fmt.Sprintf("[img: %s:%s]", i[e.ImageID].Repository, e.ImageTag)
}

type userLabeler map[string]coder.User

func (u userLabeler) label(e coder.Workspace) string {
	return fmt.Sprintf("[user: %s]", u[e.UserID].Email)
}

type providerLabeler map[string]coder.KubernetesProvider

func (p providerLabeler) label(e coder.Workspace) string {
	return fmt.Sprintf("[provider: %s]", p[e.ResourcePoolID].Name)
}

func aggregateWorkspaceResources(workspaces []coder.Workspace) resources {
	var aggregate resources
	for _, e := range workspaces {
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
		"[cpu: %.1f]\t[mem: %.1f GB]",
		a.cpuAllocation, a.memAllocation,
	)
}

//nolint:unparam
// truncate the given string and replace the removed chars with some replacement (ex: "...").
func truncate(str string, max int, replace string) string {
	if len(str) <= max {
		return str
	}
	return str[:max+1] + replace
}
