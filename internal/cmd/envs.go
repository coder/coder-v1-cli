package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sync/atomic"

	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/coder-cli/internal/clog"
	"cdr.dev/coder-cli/internal/x/xtabwriter"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"golang.org/x/xerrors"
)

const (
	defaultOrg              = "default"
	defaultImgTag           = "latest"
	defaultCPUCores float32 = 1
	defaultMemGB    float32 = 1
	defaultDiskGB           = 10
	defaultGPUs             = 0
)

func envsCommand() *cobra.Command {
	var outputFmt string
	var user string
	cmd := &cobra.Command{
		Use:   "envs",
		Short: "Interact with Coder environments",
		Long:  "Perform operations on the Coder environments owned by the active user.",
	}
	cmd.PersistentFlags().StringVar(&user, "user", coder.Me, "Specify the user whose resources to target")

	lsCmd := &cobra.Command{
		Use:   "ls",
		Short: "list all environments owned by the active user",
		Long:  "List all Coder environments owned by the active user.",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return err
			}
			envs, err := getEnvs(cmd.Context(), client, user)
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
	lsCmd.Flags().StringVarP(&outputFmt, "output", "o", "human", "human | json")
	cmd.AddCommand(lsCmd)
	cmd.AddCommand(stopEnvCommand(&user))
	cmd.AddCommand(watchBuildLogCommand())
	cmd.AddCommand(rebuildEnvCommand())
	cmd.AddCommand(createEnvCommand())
	return cmd
}

func stopEnvCommand(user *string) *cobra.Command {
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
							clog.Cause(err.Error()), clog.BlankLine,
							clog.Hint("current environment status is %q", env.LatestStat.ContainerStatus),
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

func createEnvCommand() *cobra.Command {
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
				return xerrors.New("image id unset")
			}
			// ExactArgs(1) ensures our name value can't panic on an out of bounds.
			createReq := &coder.CreateEnvironmentRequest{
				Name:     args[0],
				ImageID:  img,
				ImageTag: tag,
			}
			// We're explicitly ignoring errors for these because all of these flags
			// have a non-zero-value default value set already.
			createReq.CPUCores, _ = cmd.Flags().GetFloat32("cpu")
			createReq.MemoryGB, _ = cmd.Flags().GetFloat32("memory")
			createReq.DiskGB, _ = cmd.Flags().GetInt("disk")
			createReq.GPUs, _ = cmd.Flags().GetInt("gpus")

			client, err := newClient()
			if err != nil {
				return err
			}

			env, err := client.CreateEnvironment(cmd.Context(), org, *createReq)
			if err != nil {
				return xerrors.Errorf("create environment: %w", err)
			}

			clog.LogSuccess(
				"creating environment...",
				clog.BlankLine,
				clog.Tip(`run "coder envs watch-build %q" to trail the build logs`, args[0]),
			)

			if follow {
				if err := trailBuildLogs(cmd.Context(), client, env.ID); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&org, "org", "o", defaultOrg, "ID of the organization the environment should be created under.")
	cmd.Flags().StringVarP(&tag, "tag", "t", defaultImgTag, "tag of the image the environment will be based off of.")
	cmd.Flags().Float32P("cpu", "c", defaultCPUCores, "number of cpu cores the environment should be provisioned with.")
	cmd.Flags().Float32P("memory", "m", defaultMemGB, "GB of RAM an environment should be provisioned with.")
	cmd.Flags().IntP("disk", "d", defaultDiskGB, "GB of disk storage an environment should be provisioned with.")
	cmd.Flags().IntP("gpus", "g", defaultGPUs, "number GPUs an environment should be provisioned with.")
	cmd.Flags().StringVarP(&img, "image", "i", "", "ID of the image to base the environment off of.")
	cmd.Flags().BoolVar(&follow, "follow", false, "follow buildlog after initiating rebuild")
	cmd.MarkFlagRequired("image")
	return cmd
}
