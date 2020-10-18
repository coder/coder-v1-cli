## coder users ls

list all user accounts

### Synopsis

list all user accounts

```
coder users ls [flags]
```

### Examples

```
coder users ls -o json
coder users ls -o json | jq .[] | jq -r .email
```

### Options

```
  -h, --help            help for ls
  -o, --output string   human | json (default "human")
```

### Options inherited from parent commands

```
  -v, --verbose   show verbose output
```

### SEE ALSO

* [coder users](coder_users.md)	 - Interact with Coder user accounts
