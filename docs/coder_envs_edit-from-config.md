## coder envs edit-from-config

change the template an environment is tracking

### Synopsis

Edit an existing Coder environment using a Workspaces As Code template.

```
coder envs edit-from-config [flags]
```

### Examples

```
# edit a new environment from git repository
coder envs edit-from-config dev-env --repo-url https://github.com/cdr/m --ref my-branch
coder envs edit-from-config dev-env -f coder.yaml
```

### Options

```
  -f, --filepath string   path to local template file.
      --follow            follow buildlog after initiating rebuild
  -h, --help              help for edit-from-config
```

### Options inherited from parent commands

```
  -v, --verbose   show verbose output
```

### SEE ALSO

* [coder envs](coder_envs.md)	 - Interact with Coder environments

