## coder tls disable

Delete TLS certificates from Coder, effectively disabling https access

```
coder tls disable [flags]
```

### Examples

```
tls disable
```

### Options

```
  -f, --force   For Let's Encrypt certificates only: delete the certificate from Coder even if revocation at the Certificate Authority fails
  -h, --help    help for disable
```

### Options inherited from parent commands

```
  -v, --verbose   show verbose output
```

### SEE ALSO

* [coder tls](coder_tls.md)	 - Manage Coder TLS configuration

