## coder workspaces stop

stop Coder workspaces by name

### Synopsis

Stop Coder workspaces by name

```
coder workspaces stop [...workspace_names] [flags]
```

### Examples

```
coder workspaces stop front-end-workspace
coder workspaces stop front-end-workspace backend-workspace

# stop all of your workspaces
coder workspaces ls -o json | jq -c '.[].name' | xargs coder workspaces stop

# stop all workspaces for a given user
coder workspaces --user charlie@coder.com ls -o json \
	| jq -c '.[].name' \
	| xargs coder workspaces --user charlie@coder.com stop
```

### Options

```
  -h, --help          help for stop
      --user string   Specify the user whose resources to target (default "me")
```

### Options inherited from parent commands

```
  -v, --verbose   show verbose output
```

### SEE ALSO

* [coder workspaces](coder_workspaces.md)	 - Interact with Coder workspaces

