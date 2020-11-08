## coder tags create

add an image tag

### Synopsis

allow users to create environments with this image tag

```
coder tags create [tag] [flags]
```

### Examples

```
coder tags create latest --image ubuntu --org default
```

### Options

```
      --default        make this tag the default for its image
  -h, --help           help for create
  -i, --image string   image name
  -o, --org string     organization name
```

### Options inherited from parent commands

```
  -v, --verbose   show verbose output
```

### SEE ALSO

* [coder tags](coder_tags.md)	 - operate on Coder image tags

