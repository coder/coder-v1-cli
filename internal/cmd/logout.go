package cmd

import (
	"os"

	"cdr.dev/coder-cli/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/xerrors"

	"go.coder.com/flog"
)

func makeLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove local authentication credentials if any exist",
		RunE:  logout,
	}
}

func logout(_ *cobra.Command, _ []string) error {
	err := config.Session.Delete()
	if err != nil {
		if os.IsNotExist(err) {
			flog.Info("no active session")
			return nil
		}
		return xerrors.Errorf("delete session: %w", err)
	}
	flog.Success("logged out")
	return nil
}
