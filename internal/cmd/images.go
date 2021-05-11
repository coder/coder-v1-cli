package cmd

import (
	"encoding/json"

	"github.com/spf13/cobra"
	"golang.org/x/xerrors"

	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/coder-cli/pkg/clog"
	"cdr.dev/coder-cli/pkg/tablewriter"
)

func imgsCmd() *cobra.Command {
	var user string

	cmd := &cobra.Command{
		Use:   "images",
		Short: "Manage Coder images",
		Long:  "Manage existing images and/or import new ones.",
	}

	cmd.PersistentFlags().StringVar(&user, "user", coder.Me, "Specifies the user by email")
	cmd.AddCommand(lsImgsCommand(&user))
	return cmd
}

func lsImgsCommand(user *string) *cobra.Command {
	var (
		orgName   string
		outputFmt string
	)

	cmd := &cobra.Command{
		Use:   "ls",
		Short: "list all images available to the active user",
		Long:  "List all Coder images available to the active user.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			client, err := newClient(ctx, true)
			if err != nil {
				return err
			}

			imgs, err := getImgs(ctx, client,
				getImgsConf{
					email:   *user,
					orgName: orgName,
				},
			)

			if err != nil {
				return err
			}

			if len(imgs) < 1 {
				clog.LogInfo("no images found")
				imgs = []coder.Image{} // ensures that json output still marshals
			}

			switch outputFmt {
			case jsonOutput:
				enc := json.NewEncoder(cmd.OutOrStdout())
				// pretty print the json
				enc.SetIndent("", "\t")

				if err := enc.Encode(imgs); err != nil {
					return xerrors.Errorf("write images as JSON: %w", err)
				}
				return nil
			case humanOutput:
				err = tablewriter.WriteTable(cmd.OutOrStdout(), len(imgs), func(i int) interface{} {
					return imgs[i]
				})
				if err != nil {
					return xerrors.Errorf("write table: %w", err)
				}
				return nil
			default:
				return xerrors.Errorf("%q is not a supported value for --output", outputFmt)
			}
		},
	}
	cmd.Flags().StringVar(&orgName, "org", "", "organization name")
	cmd.Flags().StringVar(&outputFmt, "output", humanOutput, "human | json")
	return cmd
}
