## coder sh

Open a shell and execute commands in a Coder environment

### Synopsis

Execute a remote command on the environment
If no command is specified, the default shell is opened.
If the command is run in an interactive shell, a user prompt will occur if the environment needs to be rebuilt.

```
coder sh [environment_name] [<command [args...]>] [flags]
```

### Examples

```
coder sh backend-env
coder sh front-end-dev cat ~/config.json
```

### Options

```
  -h, --help   help for sh
```

### Options inherited from parent commands

```
  -v, --verbose   show verbose output
```

### SEE ALSO

* [coder](coder.md)	 - coder provides a CLI for working with an existing Coder Enterprise installation

