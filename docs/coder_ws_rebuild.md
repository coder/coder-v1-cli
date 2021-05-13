## coder ws rebuild

rebuild a Coder workspace

```
coder ws rebuild [workspace_name] [flags]
```

### Examples

```
coder ws rebuild front-end-workspace --follow
coder ws rebuild backend-workspace --force
```

### Options

```
      --follow        follow build log after initiating rebuild
      --force         force rebuild without showing a confirmation prompt
  -h, --help          help for rebuild
      --user string   Specify the user whose resources to target (default "me")
```

### Options inherited from parent commands

```
  -v, --verbose   show verbose output
```

### SEE ALSO

* [coder ws](coder_ws.md)	 - Interact with Coder workspaces

