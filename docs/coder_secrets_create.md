## coder secrets create

Create a new secret

### Synopsis

Create a new secret object to store application secrets and access them securely from within your environments.

```
coder secrets create [secret_name] [flags]
```

### Examples

```
coder secrets create mysql-password --from-literal 123password
coder secrets create mysql-password --from-prompt
coder secrets create aws-credentials --from-file ./credentials.json
```

### Options

```
      --description string    a description of the secret
      --from-file string      a file from which to read the value of the secret
      --from-literal string   the value of the secret
      --from-prompt           enter the secret value through a terminal prompt
  -h, --help                  help for create
```

### Options inherited from parent commands

```
      --user string   Specify the user whose resources to target (default "me")
  -v, --verbose       show verbose output
```

### SEE ALSO

* [coder secrets](coder_secrets.md)	 - Interact with Coder Secrets
