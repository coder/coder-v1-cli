## coder workspaces create-from-config

create a new workspace from a template

### Synopsis

Create a new Coder workspace using a workspace template.

```
coder workspaces create-from-config [flags]
```

### Examples

```
# create a new workspace from git repository
coder workspaces create-from-config --name="dev-env" --repo-url https://github.com/cdr/m --ref my-branch
coder workspaces create-from-config --name="dev-env" --filepath coder.yaml
```

### Options

```
  -f, --filepath string   path to local template file.
      --follow            follow buildlog after initiating rebuild
  -h, --help              help for create-from-config
      --name string       name of the workspace to be created
  -o, --org string        name of the organization the workspace should be created under.
      --provider string   name of Workspace Provider with which to create the workspace
      --ref string        git reference to pull template from. May be a branch, tag, or commit hash. (default "master")
  -r, --repo-url string   URL of the git repository to pull the config from. Config file must live in '.coder/coder.yaml'.
```

### Options inherited from parent commands

```
  -v, --verbose   show verbose output
```

### SEE ALSO

* [coder workspaces](coder_workspaces.md)	 - Interact with Coder workspaces

