package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"

	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/coder-cli/internal/coderutil"
	"cdr.dev/coder-cli/internal/config"
	"cdr.dev/coder-cli/internal/x/xcobra"
	"cdr.dev/coder-cli/pkg/clog"
	"cdr.dev/coder-cli/pkg/tablewriter"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"golang.org/x/xerrors"
)

const defaultImgTag = "latest"

func envsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "envs",
		Short: "Interact with Coder environments",
		Long:  "Perform operations on the Coder environments owned by the active user.",
	}

	cmd.AddCommand(
		lsEnvsCommand(),
		stopEnvsCmd(),
		rmEnvsCmd(),
		watchBuildLogCommand(),
		rebuildEnvCommand(),
		createEnvCmd(),
		createEnvFromRepoCmd(),
		editEnvCmd(),
	)
	return cmd
}

const (
	humanOutput = "human"
	jsonOutput  = "json"
)

func lsEnvsCommand() *cobra.Command {
	var (
		outputFmt string
		user      string
		provider  string
	)

	cmd := &cobra.Command{
		Use:   "ls",
		Short: "list all environments owned by the active user",
		Long:  "List all Coder environments owned by the active user.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := newClient(ctx)
			if err != nil {
				return err
			}
			envs, err := getEnvs(ctx, client, user)
			if err != nil {
				return err
			}
			if provider != "" {
				envs, err = getEnvsByProvider(ctx, client, provider, user)
				if err != nil {
					return err
				}
			}
			if len(envs) < 1 {
				clog.LogInfo("no environments found")
				envs = []coder.Environment{} // ensures that json output still marshals
			}

			switch outputFmt {
			case humanOutput:
				envs, err := coderutil.EnvsHumanTable(ctx, client, envs)
				if err != nil {
					return err
				}
				err = tablewriter.WriteTable(cmd.OutOrStdout(), len(envs), func(i int) interface{} {
					return envs[i]
				})
				if err != nil {
					return xerrors.Errorf("write table: %w", err)
				}
			case jsonOutput:
				err := json.NewEncoder(cmd.OutOrStdout()).Encode(envs)
				if err != nil {
					return xerrors.Errorf("write environments as JSON: %w", err)
				}
			default:
				return xerrors.Errorf("unknown --output value %q", outputFmt)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&user, "user", coder.Me, "Specify the user whose resources to target")
	cmd.Flags().StringVarP(&outputFmt, "output", "o", humanOutput, "human | json")
	cmd.Flags().StringVarP(&provider, "provider", "p", "", "Filter environments by a particular workspace provider name.")

	return cmd
}

func stopEnvsCmd() *cobra.Command {
	var user string
	cmd := &cobra.Command{
		Use:   "stop [...environment_names]",
		Short: "stop Coder environments by name",
		Long:  "Stop Coder environments by name",
		Example: `coder envs stop front-end-env
coder envs stop front-end-env backend-env

# stop all of your environments
coder envs ls -o json | jq -c '.[].name' | xargs coder envs stop

# stop all environments for a given user
coder envs --user charlie@coder.com ls -o json \
	| jq -c '.[].name' \
	| xargs coder envs --user charlie@coder.com stop`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := newClient(ctx)
			if err != nil {
				return xerrors.Errorf("new client: %w", err)
			}

			egroup := clog.LoggedErrGroup()
			for _, envName := range args {
				envName := envName
				egroup.Go(func() error {
					env, err := findEnv(ctx, client, envName, user)
					if err != nil {
						return err
					}

					if err = client.StopEnvironment(ctx, env.ID); err != nil {
						return clog.Error(fmt.Sprintf("stop environment %q", env.Name),
							clog.Causef(err.Error()), clog.BlankLine,
							clog.Hintf("current environment status is %q", env.LatestStat.ContainerStatus),
						)
					}
					clog.LogSuccess(fmt.Sprintf("successfully stopped environment %q", envName))
					return nil
				})
			}

			return egroup.Wait()
		},
	}
	cmd.Flags().StringVar(&user, "user", coder.Me, "Specify the user whose resources to target")
	return cmd
}

func createEnvCmd() *cobra.Command {
	var (
		org             string
		cpu             float32
		memory          float32
		disk            int
		gpus            int
		img             string
		tag             string
		follow          bool
		useCVM          bool
		providerName    string
		enableAutostart bool
	)

	cmd := &cobra.Command{
		Use:   "create [environment_name]",
		Short: "create a new environment.",
		Args:  xcobra.ExactArgs(1),
		Long:  "Create a new Coder environment.",
		Example: `# create a new environment using default resource amounts
coder envs create my-new-env --image ubuntu
coder envs create my-new-powerful-env --cpu 12 --disk 100 --memory 16 --image ubuntu`,
		PreRun: func(cmd *cobra.Command, args []string) {
			autoStartInfo()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if img == "" {
				return xerrors.New("image unset")
			}

			client, err := newClient(ctx)
			if err != nil {
				return err
			}

			multiOrgMember, err := isMultiOrgMember(ctx, client, coder.Me)
			if err != nil {
				return err
			}

			if multiOrgMember && org == "" {
				return xerrors.New("org is required for multi-org members")
			}
			importedImg, err := findImg(ctx, client, findImgConf{
				email:   coder.Me,
				imgName: img,
				orgName: org,
			})
			if err != nil {
				return err
			}

			var provider *coder.KubernetesProvider
			if providerName == "" {
				provider, err = coderutil.DefaultWorkspaceProvider(ctx, client)
				if err != nil {
					return xerrors.Errorf("default workspace provider: %w", err)
				}
			} else {
				provider, err = coderutil.ProviderByName(ctx, client, providerName)
				if err != nil {
					return xerrors.Errorf("provider by name: %w", err)
				}
			}

			// ExactArgs(1) ensures our name value can't panic on an out of bounds.
			createReq := &coder.CreateEnvironmentRequest{
				Name:            args[0],
				ImageID:         importedImg.ID,
				OrgID:           importedImg.OrganizationID,
				ImageTag:        tag,
				CPUCores:        cpu,
				MemoryGB:        memory,
				DiskGB:          disk,
				GPUs:            gpus,
				UseContainerVM:  useCVM,
				ResourcePoolID:  provider.ID,
				Namespace:       provider.DefaultNamespace,
				EnableAutoStart: enableAutostart,
			}

			// if any of these defaulted to their zero value we provision
			// the create request with the imported image defaults instead.
			if createReq.CPUCores == 0 {
				createReq.CPUCores = importedImg.DefaultCPUCores
			}
			if createReq.MemoryGB == 0 {
				createReq.MemoryGB = importedImg.DefaultMemoryGB
			}
			if createReq.DiskGB == 0 {
				createReq.DiskGB = importedImg.DefaultDiskGB
			}

			env, err := client.CreateEnvironment(ctx, *createReq)
			if err != nil {
				return xerrors.Errorf("create environment: %w", err)
			}

			if follow {
				clog.LogSuccess("creating environment...")
				if err := trailBuildLogs(ctx, client, env.ID); err != nil {
					return err
				}
				return nil
			}

			clog.LogSuccess("creating environment...",
				clog.BlankLine,
				clog.Tipf(`run "coder envs watch-build %s" to trail the build logs`, env.Name),
			)
			return nil
		},
	}
	cmd.Flags().StringVarP(&org, "org", "o", "", "name of the organization the environment should be created under.")
	cmd.Flags().StringVarP(&tag, "tag", "t", defaultImgTag, "tag of the image the environment will be based off of.")
	cmd.Flags().Float32VarP(&cpu, "cpu", "c", 0, "number of cpu cores the environment should be provisioned with.")
	cmd.Flags().Float32VarP(&memory, "memory", "m", 0, "GB of RAM an environment should be provisioned with.")
	cmd.Flags().IntVarP(&disk, "disk", "d", 0, "GB of disk storage an environment should be provisioned with.")
	cmd.Flags().IntVarP(&gpus, "gpus", "g", 0, "number GPUs an environment should be provisioned with.")
	cmd.Flags().StringVarP(&img, "image", "i", "", "name of the image to base the environment off of.")
	cmd.Flags().StringVar(&providerName, "provider", "", "name of Workspace Provider with which to create the environment")
	cmd.Flags().BoolVar(&follow, "follow", false, "follow buildlog after initiating rebuild")
	cmd.Flags().BoolVar(&useCVM, "container-based-vm", false, "deploy the environment as a Container-based VM")
	cmd.Flags().BoolVar(&enableAutostart, "enable-autostart", false, "automatically start this environment at your preferred time.")
	_ = cmd.MarkFlagRequired("image")
	return cmd
}

func createEnvFromRepoCmd() *cobra.Command {
	var (
		ref          string
		repo         string
		follow       bool
		filepath     string
		org          string
		providerName string
		envName      string
	)

	cmd := &cobra.Command{
		Use:    "create-from-config",
		Short:  "create a new environment from a config file.",
		Long:   "Create a new Coder environment from a config file.",
		Hidden: true,
		Example: `# create a new environment from git repository template
coder envs create-from-config --name="dev-env" --repo-url github.com/cdr/m --ref my-branch
coder envs create-from-config --name="dev-env" -f coder.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			if envName == "" {
				return clog.Error("Must provide a environment name.",
					clog.BlankLine,
					clog.Tipf("Use --name=<env-name> to name your enviornment"),
				)
			}

			client, err := newClient(ctx)
			if err != nil {
				return err
			}

			orgs, err := getUserOrgs(ctx, client, coder.Me)
			if err != nil {
				return err
			}

			multiOrgMember := len(orgs) > 1
			if multiOrgMember && org == "" {
				return xerrors.New("org is required for multi-org members")
			}

			var userOrg *coder.Organization
			for i := range orgs {
				// Look for org by name
				if orgs[i].Name == org {
					userOrg = &orgs[i]
					break
				}
				// Or use default if the provided is blank
				if org == "" && orgs[i].Default {
					userOrg = &orgs[i]
					break
				}
			}

			if userOrg == nil {
				if org != "" {
					return xerrors.Errorf("Unable to locate org '%s'", org)
				}
				return xerrors.Errorf("Unable to locate a default organization for the user")
			}

			var rd io.Reader
			if filepath != "" {
				b, err := ioutil.ReadFile(filepath)
				if err != nil {
					return xerrors.Errorf("read local file: %w", err)
				}
				rd = bytes.NewReader(b)
			}

			req := coder.ParseTemplateRequest{
				RepoURL:  repo,
				Ref:      ref,
				Local:    rd,
				OrgID:    userOrg.ID,
				Filepath: ".coder/coder.yaml",
			}

			version, err := client.ParseTemplate(ctx, req)
			if err != nil {
				return handleAPIError(err)
			}

			provider, err := coderutil.DefaultWorkspaceProvider(ctx, client)
			if err != nil {
				return xerrors.Errorf("default workspace provider: %w", err)
			}

			env, err := client.CreateEnvironment(ctx, coder.CreateEnvironmentRequest{
				OrgID:          userOrg.ID,
				TemplateID:     version.TemplateID,
				ResourcePoolID: provider.ID,
				Namespace:      provider.DefaultNamespace,
				Name:           envName,
			})
			if err != nil {
				return handleAPIError(err)
			}

			if follow {
				clog.LogSuccess("creating environment...")
				if err := trailBuildLogs(ctx, client, env.ID); err != nil {
					return err
				}
				return nil
			}

			clog.LogSuccess("creating environment...",
				clog.BlankLine,
				clog.Tipf(`run "coder envs watch-build %s" to trail the build logs`, env.Name),
			)
			return nil
		},
	}
	cmd.Flags().StringVarP(&org, "org", "o", "", "name of the organization the environment should be created under.")
	cmd.Flags().StringVarP(&filepath, "filepath", "f", "", "path to local template file.")
	cmd.Flags().StringVarP(&ref, "ref", "", "master", "git reference to pull template from. May be a branch, tag, or commit hash.")
	cmd.Flags().StringVarP(&repo, "repo-url", "r", "", "URL of the git repository to pull the config from. Config file must live in '.coder/coder.yaml'.")
	cmd.Flags().BoolVar(&follow, "follow", false, "follow buildlog after initiating rebuild")
	cmd.Flags().StringVar(&providerName, "provider", "", "name of Workspace Provider with which to create the environment")
	cmd.Flags().StringVar(&envName, "name", "", "name of the environment to be created")
	return cmd
}

func editEnvCmd() *cobra.Command {
	var (
		org    string
		img    string
		tag    string
		cpu    float32
		memory float32
		disk   int
		gpus   int
		follow bool
		user   string
	)

	cmd := &cobra.Command{
		Use:   "edit",
		Short: "edit an existing environment and initiate a rebuild.",
		Args:  xcobra.ExactArgs(1),
		Long:  "Edit an existing environment and initate a rebuild.",
		Example: `coder envs edit back-end-env --cpu 4

coder envs edit back-end-env --disk 20`,
		PreRun: func(cmd *cobra.Command, args []string) {
			autoStartInfo()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := newClient(ctx)
			if err != nil {
				return err
			}

			envName := args[0]

			env, err := findEnv(ctx, client, envName, user)
			if err != nil {
				return err
			}

			multiOrgMember, err := isMultiOrgMember(ctx, client, user)
			if err != nil {
				return err
			}

			// if the user belongs to multiple organizations we need them to specify which one.
			if multiOrgMember && org == "" {
				return xerrors.New("org is required for multi-org members")
			}

			req, err := buildUpdateReq(ctx, client, updateConf{
				cpu:         cpu,
				memGB:       memory,
				diskGB:      disk,
				gpus:        gpus,
				environment: env,
				user:        user,
				image:       img,
				imageTag:    tag,
				orgName:     org,
			})
			if err != nil {
				return err
			}

			if err := client.EditEnvironment(ctx, env.ID, *req); err != nil {
				return xerrors.Errorf("failed to apply changes to environment %q: %w", envName, err)
			}

			if follow {
				clog.LogSuccess("applied changes to the environment, rebuilding...")
				if err := trailBuildLogs(ctx, client, env.ID); err != nil {
					return err
				}
				return nil
			}

			clog.LogSuccess("applied changes to the environment, rebuilding...",
				clog.BlankLine,
				clog.Tipf(`run "coder envs watch-build %s" to trail the build logs`, envName),
			)
			return nil
		},
	}
	cmd.Flags().StringVarP(&org, "org", "o", "", "name of the organization the environment should be created under.")
	cmd.Flags().StringVarP(&img, "image", "i", "", "name of the image you want the environment to be based off of.")
	cmd.Flags().StringVarP(&tag, "tag", "t", "latest", "image tag of the image you want to base the environment off of.")
	cmd.Flags().Float32VarP(&cpu, "cpu", "c", 0, "The number of cpu cores the environment should be provisioned with.")
	cmd.Flags().Float32VarP(&memory, "memory", "m", 0, "The amount of RAM an environment should be provisioned with.")
	cmd.Flags().IntVarP(&disk, "disk", "d", 0, "The amount of disk storage an environment should be provisioned with.")
	cmd.Flags().IntVarP(&gpus, "gpu", "g", 0, "The amount of disk storage to provision the environment with.")
	cmd.Flags().BoolVar(&follow, "follow", false, "follow buildlog after initiating rebuild")
	cmd.Flags().StringVar(&user, "user", coder.Me, "Specify the user whose resources to target")
	return cmd
}

func rmEnvsCmd() *cobra.Command {
	var (
		force bool
		user  string
	)

	cmd := &cobra.Command{
		Use:   "rm [...environment_names]",
		Short: "remove Coder environments by name",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := newClient(ctx)
			if err != nil {
				return err
			}
			if !force {
				confirm := promptui.Prompt{
					Label:     fmt.Sprintf("Delete environments %q? (all data will be lost)", args),
					IsConfirm: true,
				}
				if _, err := confirm.Run(); err != nil {
					return clog.Fatal(
						"failed to confirm deletion", clog.BlankLine,
						clog.Tipf(`use "--force" to rebuild without a confirmation prompt`),
					)
				}
			}

			egroup := clog.LoggedErrGroup()
			for _, envName := range args {
				envName := envName
				egroup.Go(func() error {
					env, err := findEnv(ctx, client, envName, user)
					if err != nil {
						return err
					}
					if err = client.DeleteEnvironment(ctx, env.ID); err != nil {
						return clog.Error(
							fmt.Sprintf(`failed to delete environment "%s"`, env.Name),
							clog.Causef(err.Error()),
						)
					}
					clog.LogSuccess(fmt.Sprintf("deleted environment %q", env.Name))
					return nil
				})
			}
			return egroup.Wait()
		},
	}
	cmd.Flags().BoolVarP(&force, "force", "f", false, "force remove the specified environments without prompting first")
	cmd.Flags().StringVar(&user, "user", coder.Me, "Specify the user whose resources to target")
	return cmd
}

type updateConf struct {
	cpu         float32
	memGB       float32
	diskGB      int
	gpus        int
	environment *coder.Environment
	user        string
	image       string
	imageTag    string
	orgName     string
}

func buildUpdateReq(ctx context.Context, client coder.Client, conf updateConf) (*coder.UpdateEnvironmentReq, error) {
	var (
		updateReq       coder.UpdateEnvironmentReq
		defaultCPUCores float32
		defaultMemGB    float32
		defaultDiskGB   int
	)

	// If this is not empty it means the user is requesting to change the environment image.
	if conf.image != "" {
		importedImg, err := findImg(ctx, client, findImgConf{
			email:   conf.user,
			imgName: conf.image,
			orgName: conf.orgName,
		})
		if err != nil {
			return nil, err
		}

		// If the user passes an image arg of the image that
		// the environment is already using, it was most likely a mistake.
		if conf.image != importedImg.Repository {
			return nil, xerrors.Errorf("environment is already using image %q", conf.image)
		}

		// Since the environment image is being changed,
		// the resource amount defaults should be changed to
		// reflect that of the default resource amounts of the new image.
		defaultCPUCores = importedImg.DefaultCPUCores
		defaultMemGB = importedImg.DefaultMemoryGB
		defaultDiskGB = importedImg.DefaultDiskGB
		updateReq.ImageID = &importedImg.ID
	} else {
		// if the environment image is not being changed, the default
		// resource amounts should reflect the default resource amounts
		// of the image the environment is already using.
		defaultCPUCores = conf.environment.CPUCores
		defaultMemGB = conf.environment.MemoryGB
		defaultDiskGB = conf.environment.DiskGB
		updateReq.ImageID = &conf.environment.ImageID
	}

	// The following logic checks to see if the user specified
	// any resource amounts for the environment that need to be changed.
	// If they did not, then we will get the zero value back
	// and should set the resource amount to the default.

	if conf.cpu == 0 {
		updateReq.CPUCores = &defaultCPUCores
	} else {
		updateReq.CPUCores = &conf.cpu
	}

	if conf.memGB == 0 {
		updateReq.MemoryGB = &defaultMemGB
	} else {
		updateReq.MemoryGB = &conf.memGB
	}

	if conf.diskGB == 0 {
		updateReq.DiskGB = &defaultDiskGB
	} else {
		updateReq.DiskGB = &conf.diskGB
	}

	// Environment disks can not be shrink so we have to overwrite this
	// if the user accidentally requests it or if the default diskGB value for a
	// newly requested image is smaller than the current amount the environment is using.
	if *updateReq.DiskGB < conf.environment.DiskGB {
		clog.LogWarn("disk can not be shrunk",
			fmt.Sprintf("keeping environment disk at %d GB", conf.environment.DiskGB),
		)
		updateReq.DiskGB = &conf.environment.DiskGB
	}

	if conf.gpus != 0 {
		updateReq.GPUs = &conf.gpus
	}

	if conf.imageTag == "" {
		// We're forced to make an alloc here because untyped string consts are not addressable.
		// i.e.  updateReq.ImageTag = &defaultImgTag results in :
		// invalid operation: cannot take address of defaultImgTag (untyped string constant "latest")
		imgTag := defaultImgTag
		updateReq.ImageTag = &imgTag
	} else {
		updateReq.ImageTag = &conf.imageTag
	}
	return &updateReq, nil
}

// TODO (Grey): Remove education in a future non-patch release.
func autoStartInfo() {
	var preferencesURI string

	accessURI, err := config.URL.Read()
	if err != nil {
		// Error is fairly benign in this case, fallback to relative URI
		preferencesURI = "/preferences"
	} else {
		preferencesURI = fmt.Sprintf("%s%s", accessURI, "/preferences?tab=autostart")
	}

	clog.LogInfo("⚡NEW: Automate daily environment startup", "Visit "+preferencesURI+" to configure your preferred time")
}
