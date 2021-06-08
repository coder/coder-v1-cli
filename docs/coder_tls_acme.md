## coder tls acme

Generate certificate via Let's Encrypt

```
coder tls acme [flags]
```

### Examples

```

tls acme --info
tls acme --email me@example.com --domains a.example.com --domains b.example.com --provider route53 --credentials AWS_ACCESS_KEY_ID=your-key-id --credentials AWS_SECRET_ACCESS_KEY=your-secret-key --credentials AWS_REGION=your-region
```

### Options

```
  -a, --agree-tos                    Agree to ACME Terms of Service
  -c, --credentials stringToString   DNS provider credentials (default [])
  -d, --domains stringArray          Domains to request certificates for
  -e, --email string                 Email to use for ACME account
  -h, --help                         help for acme
  -i, --info                         Show supported DNS providers and required credentials for each
  -p, --provider string              DNS provider hosting your domains
```

### Options inherited from parent commands

```
  -v, --verbose   show verbose output
```

### SEE ALSO

* [coder tls](coder_tls.md)	 - Manage Coder TLS configuration

