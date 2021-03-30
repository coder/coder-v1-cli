## coder envs create-from-config

create a new environment from a template

### Synopsis

Create a new Coder environment using a Workspaces As Code template.

```
coder envs create-from-config [flags]
```

### Examples

```
# create a new environment from git repository
coder envs create-from-config --name="dev-env" --repo-url https://github.com/cdr/m --ref my-branch
coder envs create-from-config --name="dev-env" -f coder.yaml
```

### Options

```
  -f, --filepath string   path to local template file.
      --follow            follow buildlog after initiating rebuild
  -h, --help              help for create-from-config
      --name string       name of the environment to be created
  -o, --org string        name of the organization the environment should be created under.
      --provider string   name of Workspace Provider with which to create the environment
      --ref string        git reference to pull template from. May be a branch, tag, or commit hash. (default "master")
  -r, --repo-url string   URL of the git repository to pull the config from. Config file must live in '.coder/coder.yaml'.
```

### Options inherited from parent commands

```
  -v, --verbose   show verbose output
```

### SEE ALSO

* [coder envs](coder_envs.md)	 - Interact with Coder environments

