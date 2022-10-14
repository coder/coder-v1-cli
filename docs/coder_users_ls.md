## coder users ls

list all user accounts

```console
coder users ls [flags]
```

### Examples

```console
coder users ls -o json
coder users ls -o json | jq .[] | jq -r .email
```

### Options

```console
      --after string    returns users in the list after the specified ID
      --before string   returns users in the list before the specified ID
  -h, --help            help for ls
      --limit int       maximum number of users to return (default 100)
  -o, --output string   human | json (default "human")
```

### Options inherited from parent commands

```console
      --coder-token string   API authentication token.
      --coder-url string     access url of the Coder deployment. (default "https://demo-2.cdr.dev")
  -v, --verbose   show verbose output
```

### SEE ALSO

* [coder users](coder_users.md) - Interact with Coder user accounts
