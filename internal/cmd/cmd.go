package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

func Make() *cobra.Command {
	app := &cobra.Command{
		Use:   "coder",
		Short: "coder provides a CLI for working with an existing Coder Enterprise installation",
	}

	app.AddCommand(
		makeLoginCmd(),
		makeLogoutCmd(),
		makeShellCmd(),
		makeUsersCmd(),
		makeConfigSSHCmd(),
		makeSecretsCmd(),
		makeEnvsCommand(),
		makeSyncCmd(),
		makeURLCmd(),
		completionCmd,
	)
	return app
}

// reference: https://github.com/spf13/cobra/blob/master/shell_completions.md
var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate completion script",
	Long: `To load completions:

Bash:

$ source <(coder completion bash)

# To load completions for each session, execute once:
Linux:
  $ coder completion bash > /etc/bash_completion.d/coder
MacOS:
  $ coder completion bash > /usr/local/etc/bash_completion.d/coder

Zsh:

# If shell completion is not already enabled in your environment you will need
# to enable it.  You can execute the following once:

$ echo "autoload -U compinit; compinit" >> ~/.zshrc

# To load completions for each session, execute once:
$ coder completion zsh > "${fpath[1]}/_coder"

# You will need to start a new shell for this setup to take effect.

Fish:

$ coder completion fish | source

# To load completions for each session, execute once:
$ coder completion fish > ~/.config/fish/completions/coder.fish
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.ExactValidArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			cmd.Root().GenPowerShellCompletion(os.Stdout)
		}
	},
}
