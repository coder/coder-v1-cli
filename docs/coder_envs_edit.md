## coder envs edit

edit an existing environment and initiate a rebuild.

### Synopsis

Edit an existing environment and initate a rebuild.

```
coder envs edit [flags]
```

### Examples

```
coder envs edit back-end-env --cpu 4

coder envs edit back-end-env --disk 20
```

### Options

```
  -c, --cpu float32      The number of cpu cores the environment should be provisioned with.
  -d, --disk int         The amount of disk storage an environment should be provisioned with.
      --follow           follow buildlog after initiating rebuild
  -g, --gpu int          The amount of disk storage to provision the environment with.
  -h, --help             help for edit
  -i, --image string     name of the image you want the environment to be based off of.
  -m, --memory float32   The amount of RAM an environment should be provisioned with.
  -o, --org string       name of the organization the environment should be created under.
  -t, --tag string       image tag of the image you want to base the environment off of. (default "latest")
      --user string      Specify the user whose resources to target (default "me")
```

### Options inherited from parent commands

```
  -v, --verbose   show verbose output
```

### SEE ALSO

* [coder envs](coder_envs.md)	 - Interact with Coder environments

