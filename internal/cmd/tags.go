package cmd

import (
	"encoding/json"
	"os"

	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/coder-cli/pkg/clog"
	"cdr.dev/coder-cli/pkg/tablewriter"
	"github.com/spf13/cobra"
	"golang.org/x/xerrors"
)

func tagsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tags",
		Short: "operate on Coder image tags",
	}

	cmd.AddCommand(
		tagsLsCmd(),
		tagsCreateCmd(),
		tagsRmCmd(),
	)
	return cmd
}

func tagsCreateCmd() *cobra.Command {
	var (
		orgName    string
		imageName  string
		defaultTag bool
	)
	cmd := &cobra.Command{
		Use:     "create [tag]",
		Example: `coder tags create latest --image ubuntu --org default`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := newClient(ctx)
			if err != nil {
				return err
			}
			img, err := findImg(ctx, client, findImgConf{
				orgName: orgName,
				imgName: imageName,
				email:   coder.Me,
			})
			if err != nil {
				return xerrors.Errorf("find image: %w", err)
			}

			_, err = client.CreateImageTag(ctx, img.ID, coder.CreateImageTagReq{
				Tag:     args[0],
				Default: defaultTag,
			})
			if err != nil {
				return xerrors.Errorf("create image tag: %w", err)
			}
			clog.LogSuccess("created new tag")

			return nil
		},
	}

	cmd.Flags().StringVarP(&imageName, "image", "i", "", "image name")
	cmd.Flags().StringVarP(&orgName, "org", "o", "", "organization name")
	cmd.Flags().BoolVar(&defaultTag, "default", false, "make this tag the default for its image")
	_ = cmd.MarkFlagRequired("org")
	_ = cmd.MarkFlagRequired("image")
	return cmd
}

func tagsLsCmd() *cobra.Command {
	var (
		orgName   string
		imageName string
		outputFmt string
	)
	cmd := &cobra.Command{
		Use:     "ls",
		Example: `coder tags ls --image ubuntu --org default --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := newClient(ctx)
			if err != nil {
				return err
			}

			img, err := findImg(ctx, client, findImgConf{
				email:   coder.Me,
				orgName: orgName,
				imgName: imageName,
			})
			if err != nil {
				return err
			}

			tags, err := client.ImageTags(ctx, img.ID)
			if err != nil {
				return err
			}

			switch outputFmt {
			case humanOutput:
				err = tablewriter.WriteTable(len(tags), func(i int) interface{} { return tags[i] })
				if err != nil {
					return err
				}
			case jsonOutput:
				err := json.NewEncoder(os.Stdout).Encode(tags)
				if err != nil {
					return err
				}
			default:
				return clog.Error("unknown --output value")
			}

			return nil
		},
	}
	cmd.Flags().StringVar(&orgName, "org", "", "organization by name")
	cmd.Flags().StringVarP(&imageName, "image", "i", "", "image by name")
	cmd.Flags().StringVar(&outputFmt, "output", humanOutput, "output format (human|json)")
	_ = cmd.MarkFlagRequired("image")
	_ = cmd.MarkFlagRequired("org")
	return cmd
}

func tagsRmCmd() *cobra.Command {
	var (
		imageName string
		orgName   string
	)
	cmd := &cobra.Command{
		Use:     "rm [tag]",
		Example: `coder tags rm latest --image ubuntu --org default`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := newClient(ctx)
			if err != nil {
				return err
			}

			img, err := findImg(ctx, client, findImgConf{
				email:   coder.Me,
				imgName: imageName,
				orgName: orgName,
			})
			if err != nil {
				return err
			}

			if err = client.DeleteImageTag(ctx, img.ID, args[0]); err != nil {
				return err
			}
			clog.LogSuccess("removed tag")

			return nil
		},
	}
	cmd.Flags().StringVarP(&orgName, "org", "o", "", "organization by name")
	cmd.Flags().StringVarP(&imageName, "image", "i", "", "image by name")
	_ = cmd.MarkFlagRequired("image")
	_ = cmd.MarkFlagRequired("org")
	return cmd
}
