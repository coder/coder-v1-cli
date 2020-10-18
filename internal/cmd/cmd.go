package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

// verbose is a global flag for specifying that a command should give verbose output
var verbose bool = false

// Make constructs the "coder" root command
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
		makeResourceCmd(),
		completionCmd,
		genDocs(app),
	)
	app.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "show verbose output")
	return app
}

func genDocs(rootCmd *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:     "gen-docs [dir_path]",
		Short:   "Generate a markdown documentation tree for the root command.",
		Example: "coder gen-docs ./docs",
		Args:    cobra.ExactArgs(1),
		Hidden:  true,
		RunE: func(_ *cobra.Command, args []string) error {
			return doc.GenMarkdownTree(rootCmd, args[0])
		},
	}
}

// reference: https://github.com/spf13/cobra/blob/master/shell_completions.md
var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate completion script",
	Example: `coder completion fish > ~/.config/fish/completions/coder.fish
coder completion zsh > "${fpath[1]}/_coder"

Linux:
  $ coder completion bash > /etc/bash_completion.d/coder
MacOS:
  $ coder completion bash > /usr/local/etc/bash_completion.d/coder`,
	Long: `To load completions:

Bash:

$ source <(coder completion bash)

To load completions for each session, execute once:
Linux:
  $ coder completion bash > /etc/bash_completion.d/coder
MacOS:
  $ coder completion bash > /usr/local/etc/bash_completion.d/coder

Zsh:

If shell completion is not already enabled in your environment you will need
to enable it.  You can execute the following once:

$ echo "autoload -U compinit; compinit" >> ~/.zshrc

To load completions for each session, execute once:
$ coder completion zsh > "${fpath[1]}/_coder"

You will need to start a new shell for this setup to take effect.

Fish:

$ coder completion fish | source

To load completions for each session, execute once:
$ coder completion fish > ~/.config/fish/completions/coder.fish
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.ExactValidArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			_ = cmd.Root().GenBashCompletion(os.Stdout) // Best effort.
		case "zsh":
			_ = cmd.Root().GenZshCompletion(os.Stdout) // Best effort.
		case "fish":
			_ = cmd.Root().GenFishCompletion(os.Stdout, true) // Best effort.
		case "powershell":
			_ = cmd.Root().GenPowerShellCompletion(os.Stdout) // Best effort.
		}
	},
}
