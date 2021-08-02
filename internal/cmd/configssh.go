package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/cli/safeexec"
	"github.com/spf13/cobra"
	"golang.org/x/xerrors"

	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/coder-cli/internal/coderutil"
	"cdr.dev/coder-cli/pkg/clog"
)

const sshStartToken = "# ------------START-CODER-ENTERPRISE-----------"
const sshStartMessage = `# The following has been auto-generated by "coder config-ssh"
# to make accessing your Coder workspaces easier.
#
# To remove this blob, run:
#
#    coder config-ssh --remove
#
# You should not hand-edit this section, unless you are deleting it.`
const sshEndToken = "# ------------END-CODER-ENTERPRISE------------"

func configSSHCmd() *cobra.Command {
	var (
		configpath string
		remove     = false
	)

	cmd := &cobra.Command{
		Use:   "config-ssh",
		Short: "Configure SSH to access Coder workspaces",
		Long:  "Inject the proper OpenSSH configuration into your local SSH config file.",
		RunE:  configSSH(&configpath, &remove),
	}
	cmd.Flags().StringVar(&configpath, "filepath", filepath.Join("~", ".ssh", "config"), "override the default path of your ssh config file")
	cmd.Flags().BoolVar(&remove, "remove", false, "remove the auto-generated Coder ssh config")

	return cmd
}

func configSSH(configpath *string, remove *bool) func(cmd *cobra.Command, _ []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		ctx := cmd.Context()
		usr, err := user.Current()
		if err != nil {
			return xerrors.Errorf("get user home directory: %w", err)
		}

		privateKeyFilepath := filepath.Join(usr.HomeDir, ".ssh", "coder_enterprise")

		if strings.HasPrefix(*configpath, "~") {
			*configpath = strings.Replace(*configpath, "~", usr.HomeDir, 1)
		}

		currentConfig, err := readStr(*configpath)
		if os.IsNotExist(err) {
			// SSH configs are not always already there.
			currentConfig = ""
		} else if err != nil {
			return xerrors.Errorf("read ssh config file %q: %w", *configpath, err)
		}

		currentConfig, didRemoveConfig := removeOldConfig(currentConfig)
		if *remove {
			if !didRemoveConfig {
				return xerrors.Errorf("the Coder ssh configuration section could not be safely deleted or does not exist")
			}

			err = writeStr(*configpath, currentConfig)
			if err != nil {
				return xerrors.Errorf("write to ssh config file %q: %s", *configpath, err)
			}
			_ = os.Remove(privateKeyFilepath)

			return nil
		}

		client, err := newClient(ctx, true)
		if err != nil {
			return err
		}

		user, err := client.Me(ctx)
		if err != nil {
			return xerrors.Errorf("fetch username: %w", err)
		}

		workspaces, err := getWorkspaces(ctx, client, coder.Me)
		if err != nil {
			return err
		}
		if len(workspaces) < 1 {
			return xerrors.New("no workspaces found")
		}

		workspacesWithProviders, err := coderutil.WorkspacesWithProvider(ctx, client, workspaces)
		if err != nil {
			return xerrors.Errorf("resolve workspace workspace providers: %w", err)
		}

		if !sshAvailable(workspacesWithProviders) {
			return xerrors.New("SSH is disabled or not available for any workspaces in your Coder deployment.")
		}

		binPath, err := binPath()
		if err != nil {
			return xerrors.Errorf("Failed to get executable path: %w", err)
		}

		newConfig := makeNewConfigs(binPath, user.Username, workspacesWithProviders, privateKeyFilepath)

		err = os.MkdirAll(filepath.Dir(*configpath), os.ModePerm)
		if err != nil {
			return xerrors.Errorf("make configuration directory: %w", err)
		}
		err = writeStr(*configpath, currentConfig+newConfig)
		if err != nil {
			return xerrors.Errorf("write new configurations to ssh config file %q: %w", *configpath, err)
		}
		err = writeSSHKey(ctx, client, privateKeyFilepath)
		if err != nil {
			if !xerrors.Is(err, os.ErrPermission) {
				return xerrors.Errorf("write ssh key: %w", err)
			}
			fmt.Printf("Your private ssh key already exists at \"%s\"\nYou may need to remove the existing private key file and re-run this command\n\n", privateKeyFilepath)
		} else {
			fmt.Printf("Your private ssh key was written to \"%s\"\n", privateKeyFilepath)
		}

		writeSSHUXState(ctx, client, user.ID, workspaces)
		fmt.Printf("An auto-generated ssh config was written to \"%s\"\n", *configpath)
		fmt.Println("You should now be able to ssh into your workspace")
		fmt.Printf("For example, try running\n\n\t$ ssh coder.%s\n\n", workspaces[0].Name)
		return nil
	}
}

// binPath returns the path to the coder binary suitable for use in ssh
// ProxyCommand.
func binPath() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", xerrors.Errorf("get executable path: %w", err)
	}

	// On Windows, the coder-cli executable must be in $PATH for both Msys2/Git
	// Bash and OpenSSH for Windows (used by Powershell and VS Code) to function
	// correctly. Check if the current executable is in $PATH, and warn the user
	// if it isn't.
	if runtime.GOOS == "windows" {
		binName := filepath.Base(exePath)

		// We use safeexec instead of os/exec because os/exec returns paths in
		// the current working directory, which we will run into very often when
		// looking for our own path.
		pathPath, err := safeexec.LookPath(binName)
		if err != nil {
			clog.LogWarn(
				"The current executable is not in $PATH.",
				"This may lead to problems connecting to your workspace via SSH.",
				fmt.Sprintf("Please move %q to a location in your $PATH (such as System32) and run `%s config-ssh` again.", binName, binName),
			)
			// Return the exePath so SSH at least works outside of Msys2.
			return exePath, nil
		}

		// Warn the user if the current executable is not the same as the one in
		// $PATH.
		if filepath.Clean(pathPath) != filepath.Clean(exePath) {
			clog.LogWarn(
				"The current executable path does not match the executable path found in $PATH.",
				"This may lead to problems connecting to your workspace via SSH.",
				fmt.Sprintf("\t Current executable path: %q", exePath),
				fmt.Sprintf("\tExecutable path in $PATH: %q", pathPath),
			)
		}

		return binName, nil
	}

	// On platforms other than Windows we can use the full path to the binary.
	return exePath, nil
}

// removeOldConfig removes the old ssh configuration from the user's sshconfig.
// Returns true if the config was modified.
func removeOldConfig(config string) (string, bool) {
	startIndex := strings.Index(config, sshStartToken)
	endIndex := strings.Index(config, sshEndToken)

	if startIndex == -1 || endIndex == -1 {
		return config, false
	}
	if startIndex == 0 {
		return config[endIndex+len(sshEndToken)+1:], true
	}
	return config[:startIndex-1] + config[endIndex+len(sshEndToken)+1:], true
}

// sshAvailable returns true if SSH is available for at least one workspace.
func sshAvailable(workspaces []coderutil.WorkspaceWithWorkspaceProvider) bool {
	for _, workspace := range workspaces {
		if workspace.WorkspaceProvider.SSHEnabled {
			return true
		}
	}
	return false
}

func writeSSHKey(ctx context.Context, client coder.Client, privateKeyPath string) error {
	key, err := client.SSHKey(ctx)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(privateKeyPath, []byte(key.PrivateKey), 0600)
}

func makeNewConfigs(binPath, userName string, workspaces []coderutil.WorkspaceWithWorkspaceProvider, privateKeyFilepath string) string {
	newConfig := fmt.Sprintf("\n%s\n%s\n\n", sshStartToken, sshStartMessage)

	sort.Slice(workspaces, func(i, j int) bool { return workspaces[i].Workspace.Name < workspaces[j].Workspace.Name })

	for _, workspace := range workspaces {
		if !workspace.WorkspaceProvider.SSHEnabled {
			clog.LogWarn(fmt.Sprintf("SSH is not enabled for workspace provider %q", workspace.WorkspaceProvider.Name),
				clog.BlankLine,
				clog.Tipf("ask an infrastructure administrator to enable SSH for this workspace provider"),
			)
			continue
		}

		newConfig += makeSSHConfig(binPath, userName, workspace.Workspace.Name, privateKeyFilepath)
	}
	newConfig += fmt.Sprintf("\n%s\n", sshEndToken)

	return newConfig
}

func makeSSHConfig(binPath, userName, workspaceName, privateKeyFilepath string) string {
	entry := fmt.Sprintf(
		`Host coder.%s
   HostName coder.%s
   ProxyCommand "%s" tunnel %s 12213 stdio
   StrictHostKeyChecking no
   ConnectTimeout=0
   IdentitiesOnly yes
   IdentityFile="%s"
`, workspaceName, workspaceName, binPath, workspaceName, privateKeyFilepath)

	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		entry += `   ControlMaster auto
   ControlPath ~/.ssh/.connection-%r@%h:%p
   ControlPersist 600
`
	}

	return entry
}

func writeStr(filename, data string) error {
	return ioutil.WriteFile(filename, []byte(data), 0777)
}

func readStr(filename string) (string, error) {
	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(contents), nil
}

func writeSSHUXState(ctx context.Context, client coder.Client, userID string, workspaces []coder.Workspace) {
	// Create a map of workspace.ID -> true to indicate to the web client that all
	// current workspaces have SSH configured
	cliSSHConfigured := make(map[string]bool)
	for _, workspace := range workspaces {
		cliSSHConfigured[workspace.ID] = true
	}
	// Update UXState that coder config-ssh has been run by the currently
	// authenticated user
	err := client.UpdateUXState(ctx, userID, map[string]interface{}{"cliSSHConfigured": cliSSHConfigured})
	if err != nil {
		clog.LogWarn("The Coder web client may not recognize that you've configured SSH.")
	}
}
