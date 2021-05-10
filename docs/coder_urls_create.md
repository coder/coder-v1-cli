## coder urls create

Create a new dev URL for a workspace

```
coder urls create [workspace_name] [port] [flags]
```

### Examples

```
coder urls create my-workspace 8080 --name my-dev-url
```

### Options

```
      --access string   Set DevURL access to [private | org | authed | public] (default "private")
  -h, --help            help for create
      --name string     DevURL name
      --scheme string   Server scheme (http|https) (default "http")
```

### Options inherited from parent commands

```
  -v, --verbose   show verbose output
```

### SEE ALSO

* [coder urls](coder_urls.md)	 - Interact with environment DevURLs

