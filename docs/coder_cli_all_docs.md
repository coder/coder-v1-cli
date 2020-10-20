
## coder

coder provides a CLI for working with an existing Coder Enterprise installation

### Options

```
  -h, --help      help for coder
  -v, --verbose   show verbose output
```

### SEE ALSO

* [coder completion](#coder-completion)	 - Generate completion script
* [coder config-ssh](#coder-config-ssh)	 - Configure SSH to access Coder environments
* [coder envs](#coder-envs)	 - Interact with Coder environments
* [coder login](#coder-login)	 - Authenticate this client for future operations
* [coder logout](#coder-logout)	 - Remove local authentication credentials if any exist
* [coder secrets](#coder-secrets)	 - Interact with Coder Secrets
* [coder sh](#coder-sh)	 - Open a shell and execute commands in a Coder environment
* [coder sync](#coder-sync)	 - Establish a one way directory sync to a Coder environment
* [coder urls](#coder-urls)	 - Interact with environment DevURLs
* [coder users](#coder-users)	 - Interact with Coder user accounts


## coder users

Interact with Coder user accounts

### Options

```
  -h, --help   help for users
```

### Options inherited from parent commands

```
  -v, --verbose   show verbose output
```

### SEE ALSO

* [coder](#coder)	 - coder provides a CLI for working with an existing Coder Enterprise installation
* [coder users ls](#coder-users-ls)	 - list all user accounts


## coder users ls

list all user accounts

```
coder users ls [flags]
```

### Examples

```
coder users ls -o json
coder users ls -o json | jq .[] | jq -r .email
```

### Options

```
  -h, --help            help for ls
  -o, --output string   human | json (default "human")
```

### Options inherited from parent commands

```
  -v, --verbose   show verbose output
```

### SEE ALSO

* [coder users](#coder-users)	 - Interact with Coder user accounts


## coder urls

Interact with environment DevURLs

### Options

```
  -h, --help   help for urls
```

### Options inherited from parent commands

```
  -v, --verbose   show verbose output
```

### SEE ALSO

* [coder](#coder)	 - coder provides a CLI for working with an existing Coder Enterprise installation
* [coder urls create](#coder-urls-create)	 - Create a new devurl for an environment
* [coder urls ls](#coder-urls-ls)	 - List all DevURLs for an environment
* [coder urls rm](#coder-urls-rm)	 - Remove a dev url


## coder urls rm

Remove a dev url

```
coder urls rm [environment_name] [port] [flags]
```

### Options

```
  -h, --help   help for rm
```

### Options inherited from parent commands

```
  -v, --verbose   show verbose output
```

### SEE ALSO

* [coder urls](#coder-urls)	 - Interact with environment DevURLs


## coder urls ls

List all DevURLs for an environment

```
coder urls ls [environment_name] [flags]
```

### Options

```
  -h, --help            help for ls
  -o, --output string   human|json (default "human")
```

### Options inherited from parent commands

```
  -v, --verbose   show verbose output
```

### SEE ALSO

* [coder urls](#coder-urls)	 - Interact with environment DevURLs


## coder urls create

Create a new devurl for an environment

```
coder urls create [env_name] [port] [--access <level>] [--name <name>] [flags]
```

### Options

```
      --access string   Set DevURL access to [private | org | authed | public] (default "private")
  -h, --help            help for create
      --name string     DevURL name
```

### Options inherited from parent commands

```
  -v, --verbose   show verbose output
```

### SEE ALSO

* [coder urls](#coder-urls)	 - Interact with environment DevURLs


## coder sync

Establish a one way directory sync to a Coder environment

```
coder sync [local directory] [<env name>:<remote directory>] [flags]
```

### Options

```
  -h, --help   help for sync
      --init   do initial transfer and exit
```

### Options inherited from parent commands

```
  -v, --verbose   show verbose output
```

### SEE ALSO

* [coder](#coder)	 - coder provides a CLI for working with an existing Coder Enterprise installation


## coder sh

Open a shell and execute commands in a Coder environment

### Synopsis

Execute a remote command on the environment\nIf no command is specified, the default shell is opened.

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

* [coder](#coder)	 - coder provides a CLI for working with an existing Coder Enterprise installation


## coder secrets

Interact with Coder Secrets

### Synopsis

Interact with secrets objects owned by the active user.

### Options

```
  -h, --help          help for secrets
      --user string   Specify the user whose resources to target (default "me")
```

### Options inherited from parent commands

```
  -v, --verbose   show verbose output
```

### SEE ALSO

* [coder](#coder)	 - coder provides a CLI for working with an existing Coder Enterprise installation
* [coder secrets create](#coder-secrets-create)	 - Create a new secret
* [coder secrets ls](#coder-secrets-ls)	 - List all secrets owned by the active user
* [coder secrets rm](#coder-secrets-rm)	 - Remove one or more secrets by name
* [coder secrets view](#coder-secrets-view)	 - View a secret by name


## coder secrets view

View a secret by name

```
coder secrets view [secret_name] [flags]
```

### Examples

```
coder secrets view mysql-password
```

### Options

```
  -h, --help   help for view
```

### Options inherited from parent commands

```
      --user string   Specify the user whose resources to target (default "me")
  -v, --verbose       show verbose output
```

### SEE ALSO

* [coder secrets](#coder-secrets)	 - Interact with Coder Secrets


## coder secrets rm

Remove one or more secrets by name

```
coder secrets rm [...secret_name] [flags]
```

### Examples

```
coder secrets rm mysql-password mysql-user
```

### Options

```
  -h, --help   help for rm
```

### Options inherited from parent commands

```
      --user string   Specify the user whose resources to target (default "me")
  -v, --verbose       show verbose output
```

### SEE ALSO

* [coder secrets](#coder-secrets)	 - Interact with Coder Secrets


## coder secrets ls

List all secrets owned by the active user

```
coder secrets ls [flags]
```

### Options

```
  -h, --help   help for ls
```

### Options inherited from parent commands

```
      --user string   Specify the user whose resources to target (default "me")
  -v, --verbose       show verbose output
```

### SEE ALSO

* [coder secrets](#coder-secrets)	 - Interact with Coder Secrets


## coder secrets create

Create a new secret

### Synopsis

Create a new secret object to store application secrets and access them securely from within your environments.

```
coder secrets create [secret_name] [flags]
```

### Examples

```
coder secrets create mysql-password --from-literal 123password
coder secrets create mysql-password --from-prompt
coder secrets create aws-credentials --from-file ./credentials.json
```

### Options

```
      --description string    a description of the secret
      --from-file string      a file from which to read the value of the secret
      --from-literal string   the value of the secret
      --from-prompt           enter the secret value through a terminal prompt
  -h, --help                  help for create
```

### Options inherited from parent commands

```
      --user string   Specify the user whose resources to target (default "me")
  -v, --verbose       show verbose output
```

### SEE ALSO

* [coder secrets](#coder-secrets)	 - Interact with Coder Secrets


## coder logout

Remove local authentication credentials if any exist

```
coder logout [flags]
```

### Options

```
  -h, --help   help for logout
```

### Options inherited from parent commands

```
  -v, --verbose   show verbose output
```

### SEE ALSO

* [coder](#coder)	 - coder provides a CLI for working with an existing Coder Enterprise installation


## coder login

Authenticate this client for future operations

```
coder login [Coder Enterprise URL eg. https://my.coder.domain/] [flags]
```

### Options

```
  -h, --help   help for login
```

### Options inherited from parent commands

```
  -v, --verbose   show verbose output
```

### SEE ALSO

* [coder](#coder)	 - coder provides a CLI for working with an existing Coder Enterprise installation


## coder envs

Interact with Coder environments

### Synopsis

Perform operations on the Coder environments owned by the active user.

### Options

```
  -h, --help          help for envs
      --user string   Specify the user whose resources to target (default "me")
```

### Options inherited from parent commands

```
  -v, --verbose   show verbose output
```

### SEE ALSO

* [coder](#coder)	 - coder provides a CLI for working with an existing Coder Enterprise installation
* [coder envs ls](#coder-envs-ls)	 - list all environments owned by the active user
* [coder envs rebuild](#coder-envs-rebuild)	 - rebuild a Coder environment
* [coder envs rm](#coder-envs-rm)	 - remove Coder environments by name
* [coder envs stop](#coder-envs-stop)	 - stop Coder environments by name
* [coder envs watch-build](#coder-envs-watch-build)	 - trail the build log of a Coder environment


## coder envs watch-build

trail the build log of a Coder environment

```
coder envs watch-build [environment_name] [flags]
```

### Examples

```
coder envs watch-build front-end-env
```

### Options

```
  -h, --help   help for watch-build
```

### Options inherited from parent commands

```
      --user string   Specify the user whose resources to target (default "me")
  -v, --verbose       show verbose output
```

### SEE ALSO

* [coder envs](#coder-envs)	 - Interact with Coder environments


## coder envs stop

stop Coder environments by name

### Synopsis

Stop Coder environments by name

```
coder envs stop [...environment_names] [flags]
```

### Examples

```
coder envs stop front-end-env
coder envs stop front-end-env backend-env

# stop all of your environments
coder envs ls -o json | jq -c '.[].name' | xargs coder envs stop

# stop all environments for a given user
coder envs --user charlie@coder.com ls -o json \
	| jq -c '.[].name' \
	| xargs coder envs --user charlie@coder.com stop
```

### Options

```
  -h, --help   help for stop
```

### Options inherited from parent commands

```
      --user string   Specify the user whose resources to target (default "me")
  -v, --verbose       show verbose output
```

### SEE ALSO

* [coder envs](#coder-envs)	 - Interact with Coder environments


## coder envs rm

remove Coder environments by name

```
coder envs rm [...environment_names] [flags]
```

### Options

```
  -f, --force   force remove the specified environments without prompting first
  -h, --help    help for rm
```

### Options inherited from parent commands

```
      --user string   Specify the user whose resources to target (default "me")
  -v, --verbose       show verbose output
```

### SEE ALSO

* [coder envs](#coder-envs)	 - Interact with Coder environments


## coder envs rebuild

rebuild a Coder environment

```
coder envs rebuild [environment_name] [flags]
```

### Examples

```
coder envs rebuild front-end-env --follow
coder envs rebuild backend-env --force
```

### Options

```
      --follow   follow build log after initiating rebuild
      --force    force rebuild without showing a confirmation prompt
  -h, --help     help for rebuild
```

### Options inherited from parent commands

```
      --user string   Specify the user whose resources to target (default "me")
  -v, --verbose       show verbose output
```

### SEE ALSO

* [coder envs](#coder-envs)	 - Interact with Coder environments


## coder envs ls

list all environments owned by the active user

### Synopsis

List all Coder environments owned by the active user.

```
coder envs ls [flags]
```

### Options

```
  -h, --help            help for ls
  -o, --output string   human | json (default "human")
```

### Options inherited from parent commands

```
      --user string   Specify the user whose resources to target (default "me")
  -v, --verbose       show verbose output
```

### SEE ALSO

* [coder envs](#coder-envs)	 - Interact with Coder environments


## coder config-ssh

Configure SSH to access Coder environments

### Synopsis

Inject the proper OpenSSH configuration into your local SSH config file.

```
coder config-ssh [flags]
```

### Options

```
      --filepath string   overide the default path of your ssh config file (default "~/.ssh/config")
  -h, --help              help for config-ssh
      --remove            remove the auto-generated Coder Enterprise ssh config
```

### Options inherited from parent commands

```
  -v, --verbose   show verbose output
```

### SEE ALSO

* [coder](#coder)	 - coder provides a CLI for working with an existing Coder Enterprise installation


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

* [coder](#coder)	 - coder provides a CLI for working with an existing Coder Enterprise installation

