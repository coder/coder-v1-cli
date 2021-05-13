## coder ws create-from-config

create a new workspace from a template

### Synopsis

Create a new Coder workspace using a Workspaces As Code template.

```
coder ws create-from-config [flags]
```

### Examples

```
# create a new workspace from git repository
coder ws create-from-config --name="dev-workspace" --repo-url https://github.com/cdr/m --ref my-branch
coder ws create-from-config --name="dev-workspace" -f coder.yaml
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

* [coder ws](coder_ws.md)	 - Interact with Coder workspaces

