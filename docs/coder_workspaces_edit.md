## coder workspaces edit

edit an existing workspace and initiate a rebuild.

### Synopsis

Edit an existing workspace and initate a rebuild.

```
coder workspaces edit [flags]
```

### Examples

```
coder workspaces edit back-end-workspace --cpu 4

coder workspaces edit back-end-workspace --disk 20
```

### Options

```
  -c, --cpu float32      The number of cpu cores the workspace should be provisioned with.
  -d, --disk int         The amount of disk storage a workspace should be provisioned with.
      --follow           follow buildlog after initiating rebuild
      --force            force rebuild without showing a confirmation prompt
  -g, --gpu int          The amount of disk storage to provision the workspace with.
  -h, --help             help for edit
  -i, --image string     name of the image you want the workspace to be based off of.
  -m, --memory float32   The amount of RAM a workspace should be provisioned with.
  -o, --org string       name of the organization the workspace should be created under.
  -t, --tag string       image tag of the image you want to base the workspace off of. (default "latest")
      --user string      Specify the user whose resources to target (default "me")
```

### Options inherited from parent commands

```
  -v, --verbose   show verbose output
```

### SEE ALSO

* [coder workspaces](coder_workspaces.md)	 - Interact with Coder workspaces

