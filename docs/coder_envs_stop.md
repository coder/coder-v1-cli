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

* [coder envs](coder_envs.md)	 - Interact with Coder environments

