package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/xerrors"

	"cdr.dev/coder-cli/internal/config"
	"cdr.dev/coder-cli/pkg/clog"
)

func logoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove local authentication credentials if any exist",
		RunE:  logout,
	}
}

func logout(_ *cobra.Command, _ []string) error {
	err := config.CredentialsFile.Delete()
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
