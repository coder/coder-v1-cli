## coder envs stop

stop Coder workspaces by name

### Synopsis

Stop Coder workspaces by name

```
coder envs stop [...workspace_names] [flags]
```

### Examples

```
coder envs stop front-end-workspace
coder envs stop front-end-workspace backend-workspace

# stop all of your workspaces
coder envs ls -o json | jq -c '.[].name' | xargs coder envs stop

# stop all workspaces for a given user
coder envs --user charlie@coder.com ls -o json \
	| jq -c '.[].name' \
	| xargs coder envs --user charlie@coder.com stop
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

* [coder envs](coder_envs.md)	 - Interact with Coder workspaces
