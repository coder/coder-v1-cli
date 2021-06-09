## coder workspaces policy-template

Set workspace policy template

### Synopsis

Set workspace policy template

```
coder workspaces policy-template [flags]
```

### Options

```
      --dry-run           skip setting policy template, but view errors/warnings about how this policy template would impact existing workspaces
  -f, --filepath string   full path to local policy template file.
  -h, --help              help for policy-template
      --ref string        git reference to pull template from. May be a branch, tag, or commit hash. (default "master")
  -r, --repo-url string   URL of the git repository to pull the config from. Config file must live in '.coder/coder.yaml'.
```

### Options inherited from parent commands

```
  -v, --verbose   show verbose output
```

### SEE ALSO

* [coder workspaces](coder_workspaces.md)	 - Interact with Coder workspaces

