package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync/atomic"

	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/coder-cli/internal/clog"
	"cdr.dev/coder-cli/internal/x/xtabwriter"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"golang.org/x/xerrors"
)

const defaultImgTag = "latest"

func envsCommand() *cobra.Command {
	var user string
	cmd := &cobra.Command{
		Use:   "envs",
		Short: "Interact with Coder environments",
		Long:  "Perform operations on the Coder environments owned by the active user.",
	}
	cmd.PersistentFlags().StringVar(&user, "user", coder.Me, "Specify the user whose resources to target")

	cmd.AddCommand(
		lsEnvsCommand(&user),
		stopEnvsCommand(&user),
		rmEnvsCommand(&user),
		watchBuildLogCommand(),
		rebuildEnvCommand(),
		createEnvCommand(&user),
		editEnvCommand(&user),
	)
	return cmd
}

func lsEnvsCommand(user *string) *cobra.Command {
	var outputFmt string

	cmd := &cobra.Command{
		Use:   "ls",
		Short: "list all environments owned by the active user",
		Long:  "List all Coder environments owned by the active user.",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return err
			}
			envs, err := getEnvs(cmd.Context(), client, *user)
			if err != nil {
				return err
			}
			if len(envs) < 1 {
				clog.LogInfo("no environments found")
				return nil
			}

			switch outputFmt {
			case "human":
				err := xtabwriter.WriteTable(len(envs), func(i int) interface{} {
					return envs[i]
				})
				if err != nil {
					return xerrors.Errorf("write table: %w", err)
				}
			case "json":
				err := json.NewEncoder(os.Stdout).Encode(envs)
				if err != nil {
					return xerrors.Errorf("write environments as JSON: %w", err)
				}
			default:
				return xerrors.Errorf("unknown --output value %q", outputFmt)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&outputFmt, "output", "o", "human", "human | json")

	return cmd
}

func stopEnvsCommand(user *string) *cobra.Command {
	return &cobra.Command{
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
			client, err := newClient()
			if err != nil {
				return xerrors.Errorf("new client: %w", err)
			}

			var egroup errgroup.Group
			var fails int32
			for _, envName := range args {
				envName := envName
				egroup.Go(func() error {
					env, err := findEnv(cmd.Context(), client, envName, *user)
					if err != nil {
						atomic.AddInt32(&fails, 1)
						clog.Log(err)
						return xerrors.Errorf("find env by name: %w", err)
					}

					if err = client.StopEnvironment(cmd.Context(), env.ID); err != nil {
						atomic.AddInt32(&fails, 1)
						err = clog.Fatal(fmt.Sprintf("stop environment %q", env.Name),
							clog.Causef(err.Error()), clog.BlankLine,
							clog.Hintf("current environment status is %q", env.LatestStat.ContainerStatus),
						)
						clog.Log(err)
						return err
					}
					clog.LogSuccess(fmt.Sprintf("successfully stopped environment %q", envName))
					return nil
				})
			}

			if err = egroup.Wait(); err != nil {
				return clog.Fatal(fmt.Sprintf("%d failure(s) emitted", fails))
			}
			return nil
		},
	}
}

func createEnvCommand(user *string) *cobra.Command {
	var (
		org    string
		img    string
		tag    string
		follow bool
	)

	cmd := &cobra.Command{
		Use:   "create [environment_name]",
		Short: "create a new environment.",
		Args:  cobra.ExactArgs(1),
		// Don't unhide this command until we can pass image names instead of image id's.
		Hidden: true,
		Long:   "Create a new environment under the active user.",
		Example: `# create a new environment using default resource amounts
coder envs create --image 5f443b16-30652892427b955601330fa5 my-env-name

# create a new environment using custom resource amounts
coder envs create --cpu 4 --disk 100 --memory 8 --image 5f443b16-30652892427b955601330fa5 my-env-name`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if img == "" {
				return xerrors.New("image unset")
			}

			client, err := newClient()
			if err != nil {
				return err
			}

			multiOrgMember, err := isMultiOrgMember(cmd.Context(), client, *user)
			if err != nil {
				return err
			}

			if multiOrgMember && org == "" {
				return xerrors.New("org is required for multi-org members")
			}

			importedImg, err := findImg(cmd.Context(),
				findImgConf{
					client:  client,
					email:   *user,
					imgName: img,
					orgName: org,
				},
			)
			if err != nil {
				return err
			}

			// ExactArgs(1) ensures our name value can't panic on an out of bounds.
			createReq := &coder.CreateEnvironmentRequest{
				Name:     args[0],
				ImageID:  importedImg.ID,
				ImageTag: tag,
			}
			// We're explicitly ignoring errors for these because all we
			// need to now is if the numeric type is 0 or not.
			createReq.CPUCores, _ = cmd.Flags().GetFloat32("cpu")
			createReq.MemoryGB, _ = cmd.Flags().GetFloat32("memory")
			createReq.DiskGB, _ = cmd.Flags().GetInt("disk")
			createReq.GPUs, _ = cmd.Flags().GetInt("gpus")

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

			env, err := client.CreateEnvironment(cmd.Context(), importedImg.OrganizationID, *createReq)
			if err != nil {
				return xerrors.Errorf("create environment: %w", err)
			}

			if follow {
				clog.LogSuccess("creating environment...")
				if err := trailBuildLogs(cmd.Context(), client, env.ID); err != nil {
					return err
				}
				return nil
			}

			clog.LogSuccess("creating environment...",
				clog.BlankLine,
				clog.Tipf(`run "coder envs watch-build %q" to trail the build logs`, env.Name),
			)
			return nil
		},
	}
	cmd.Flags().StringVarP(&org, "org", "o", "", "ID of the organization the environment should be created under.")
	cmd.Flags().StringVarP(&tag, "tag", "t", defaultImgTag, "tag of the image the environment will be based off of.")
	cmd.Flags().Float32P("cpu", "c", 0, "number of cpu cores the environment should be provisioned with.")
	cmd.Flags().Float32P("memory", "m", 0, "GB of RAM an environment should be provisioned with.")
	cmd.Flags().IntP("disk", "d", 0, "GB of disk storage an environment should be provisioned with.")
	cmd.Flags().IntP("gpus", "g", 0, "number GPUs an environment should be provisioned with.")
	cmd.Flags().StringVarP(&img, "image", "i", "", "name of the image to base the environment off of.")
	cmd.Flags().BoolVar(&follow, "follow", false, "follow buildlog after initiating rebuild")
	_ = cmd.MarkFlagRequired("image")
	return cmd
}

func editEnvCommand(user *string) *cobra.Command {
	var (
		org      string
		img      string
		tag      string
		cpuCores float32
		memGB    float32
		diskGB   int
		gpus     int
		follow   bool
	)

	cmd := &cobra.Command{
		Use:    "edit",
		Short:  "edit an existing environment owned by the active user.",
		Args:   cobra.ExactArgs(1),
		Hidden: true,
		Long:   "Edit an existing environment owned by the active user.",
		Example: `coder envs edit back-end-env --cpu 4

coder envs edit back-end-env --disk 20`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return err
			}

			envName := args[0]

			env, err := findEnv(cmd.Context(), client, envName, *user)
			if err != nil {
				return err
			}

			multiOrgMember, err := isMultiOrgMember(cmd.Context(), client, *user)
			if err != nil {
				return err
			}

			// if the user belongs to multiple organizations we need them to specify which one.
			if multiOrgMember && org == "" {
				return xerrors.New("org is required for multi-org members")
			}

			cpuCores, _ = cmd.Flags().GetFloat32("cpu")
			memGB, _ = cmd.Flags().GetFloat32("memory")
			diskGB, _ = cmd.Flags().GetInt("disk")
			gpus, _ = cmd.Flags().GetInt("gpus")

			req, err := buildUpdateReq(cmd.Context(),
				updateConf{
					cpu:         cpuCores,
					memGB:       memGB,
					diskGB:      diskGB,
					gpus:        gpus,
					client:      client,
					environment: env,
					user:        user,
					image:       img,
					imageTag:    tag,
					orgName:     org,
				},
			)
			if err != nil {
				return err
			}

			if err := client.EditEnvironment(cmd.Context(), env.ID, *req); err != nil {
				return xerrors.Errorf("failed to apply changes to environment %q: %w", envName, err)
			}

			if follow {
				clog.LogSuccess("applied changes to the environment, rebuilding...")
				if err := trailBuildLogs(cmd.Context(), client, env.ID); err != nil {
					return err
				}
				return nil
			}

			clog.LogSuccess("applied changes to the environment, rebuilding...",
				clog.BlankLine,
				clog.Tipf(`run "coder envs watch-build %q" to trail the build logs`, envName),
			)
			return nil
		},
	}
	cmd.Flags().StringVarP(&org, "org", "o", "", "name of the organization the environment should be created under.")
	cmd.Flags().StringVarP(&img, "image", "i", "", "name of the image you wan't the environment to be based off of.")
	cmd.Flags().StringVarP(&tag, "tag", "t", "latest", "image tag of the image you wan't to base the environment off of.")
	cmd.Flags().Float32P("cpu", "c", cpuCores, "The number of cpu cores the environment should be provisioned with.")
	cmd.Flags().Float32P("memory", "m", memGB, "The amount of RAM an environment should be provisioned with.")
	cmd.Flags().IntP("disk", "d", diskGB, "The amount of disk storage an environment should be provisioned with.")
	cmd.Flags().IntP("gpu", "g", gpus, "The amount of disk storage to provision the environment with.")
	cmd.Flags().BoolVar(&follow, "follow", false, "follow buildlog after initiating rebuild")
	return cmd
}

func rmEnvsCommand(user *string) *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:    "rm [...environment_names]",
		Short:  "remove Coder environments by name",
		Hidden: true,
		Args:   cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := newClient()
			if err != nil {
				return err
			}
			if !force {
				confirm := promptui.Prompt{
					Label:     fmt.Sprintf("Delete environments %q? (all data will be lost)", args),
					IsConfirm: true,
				}
				if _, err := confirm.Run(); err != nil {
					return err
				}
			}

			var egroup errgroup.Group
			var failures int32
			for _, envName := range args {
				envName := envName
				egroup.Go(func() error {
					env, err := findEnv(ctx, client, envName, *user)
					if err != nil {
						atomic.AddInt32(&failures, 1)
						clog.Log(err)
						return err
					}
					if err = client.DeleteEnvironment(cmd.Context(), env.ID); err != nil {
						atomic.AddInt32(&failures, 1)
						err = clog.Error(
							fmt.Sprintf(`failed to delete environment "%s"`, env.Name),
							clog.Causef(err.Error()),
						)
						clog.Log(err)
						return err
					}
					clog.LogSuccess(fmt.Sprintf("deleted environment %q", env.Name))
					return nil
				})
			}

			if err = egroup.Wait(); err != nil {
				return xerrors.Errorf("%d failure(s) emitted", failures)
			}
			return nil
		},
	}
	cmd.Flags().BoolVarP(&force, "force", "f", false, "force remove the specified environments without prompting first")
	return cmd
}

type updateConf struct {
	cpu         float32
	memGB       float32
	diskGB      int
	gpus        int
	client      *coder.Client
	environment *coder.Environment
	user        *string
	image       string
	imageTag    string
	orgName     string
}

func buildUpdateReq(ctx context.Context, conf updateConf) (*coder.UpdateEnvironmentReq, error) {
	var (
		updateReq       coder.UpdateEnvironmentReq
		defaultCPUCores float32
		defaultMemGB    float32
		defaultDiskGB   int
	)

	// If this is not empty it means the user is requesting to change the environment image.
	if conf.image != "" {
		importedImg, err := findImg(ctx,
			findImgConf{
				client:  conf.client,
				email:   *conf.user,
				imgName: conf.image,
				orgName: conf.orgName,
			},
		)
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
