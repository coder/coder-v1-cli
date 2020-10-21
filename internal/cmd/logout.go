package cmd

import (
	"os"

	"cdr.dev/coder-cli/internal/clog"
	"cdr.dev/coder-cli/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/xerrors"
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
			clog.LogInfo("no active session")
			return nil
		}
		return xerrors.Errorf("delete session: %w", err)
	}
	clog.LogSuccess("logged out")
	return nil
}
