## coder envs rebuild

rebuild a Coder environment

```
coder envs rebuild [environment_name] [flags]
```

### Examples

```
coder envs rebuild front-end-env --follow
coder envs rebuild backend-env --force
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

* [coder envs](coder_envs.md)	 - Interact with Coder environments

