## coder ws stop

stop Coder workspaces by name

### Synopsis

Stop Coder workspaces by name

```
coder ws stop [...workspace_names] [flags]
```

### Examples

```
coder ws stop front-end-workspace
coder ws stop front-end-workspace backend-workspace

# stop all of your workspaces
coder ws ls -o json | jq -c '.[].name' | xargs coder ws stop

# stop all workspaces for a given user
coder ws --user charlie@coder.com ls -o json \
	| jq -c '.[].name' \
	| xargs coder ws --user charlie@coder.com stop
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

* [coder ws](coder_ws.md)	 - Interact with Coder workspaces

