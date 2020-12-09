## coder envs create

create a new environment.

### Synopsis

Create a new Coder environment.

```
coder envs create [environment_name] [flags]
```

### Examples

```
# create a new environment using default resource amounts
coder envs create my-new-env --image ubuntu
coder envs create my-new-powerful-env --cpu 12 --disk 100 --memory 16 --image ubuntu
```

### Options

```
      --container-vm     deploy the environment as a Container-based VM
  -c, --cpu float32      number of cpu cores the environment should be provisioned with.
  -d, --disk int         GB of disk storage an environment should be provisioned with.
      --follow           follow buildlog after initiating rebuild
  -g, --gpus int         number GPUs an environment should be provisioned with.
  -h, --help             help for create
  -i, --image string     name of the image to base the environment off of.
  -m, --memory float32   GB of RAM an environment should be provisioned with.
  -o, --org string       name of the organization the environment should be created under.
  -t, --tag string       tag of the image the environment will be based off of. (default "latest")
```

### Options inherited from parent commands

```
      --user string   Specify the user whose resources to target (default "me")
  -v, --verbose       show verbose output
```

### SEE ALSO

* [coder envs](coder_envs.md)	 - Interact with Coder environments

