## coder tls self-sign

Generate self-signed certificate for Coder

### Synopsis

Generate self-signed certificates for Coder. Self-signed certificates are automatically renewed by Coder

```
coder tls self-sign [flags]
```

### Examples

```
tls self-sign --hosts a.example.com --hosts b.example.com --hosts c.example.com
```

### Options

```
  -h, --help                help for self-sign
      --hosts stringArray   Hostnames and/or IPs to generate self-signed certificate for
```

### Options inherited from parent commands

```
  -v, --verbose   show verbose output
```

### SEE ALSO

* [coder tls](coder_tls.md)	 - Manage Coder TLS configuration

