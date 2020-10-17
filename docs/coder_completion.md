## coder completion

Generate completion script

### Synopsis

To load completions:

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


```
coder completion [bash|zsh|fish|powershell]
```

### Examples

```
coder completion fish > ~/.config/fish/completions/coder.fish
coder completion zsh > "${fpath[1]}/_coder"

Linux:
  $ coder completion bash > /etc/bash_completion.d/coder
MacOS:
  $ coder completion bash > /usr/local/etc/bash_completion.d/coder
```

### Options

```
  -h, --help   help for completion
```

### Options inherited from parent commands

```
  -v, --verbose   show verbose output
```

### SEE ALSO

* [coder](coder.md)	 - coder provides a CLI for working with an existing Coder Enterprise installation
