// +build windows

package cmd

import (
	"github.com/spf13/cobra"
	"golang.org/x/xerrors"
)

func makeSyncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync [local directory] [<env name>:<remote directory>]",
		Short: "NOT AVAILABLE ON WINDOWS – Establish a one way directory sync to a Coder environment",
		RunE: func(_ *cobra.Command, _ []string) error {
			return xerrors.Errorf(`"coder sync" is not available on Windows`)
		},
	}
	return cmd
}
