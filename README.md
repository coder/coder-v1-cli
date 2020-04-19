# coder

`coder` provides a one-way, live file sync to your Coder Enterprise environment.

It is useful in cases where you want to use an unsupported IDE with your Coder
Environment.

## Login
```shell script
$ coder login https://my-coder-enterprise.com
```

## Setting up a Sync

``
$ coder sync ~/Projects/cdr/enterprise my-env:~/enterprise
``

## Caveats

- The `coder login` flow will not work when the CLI is ran from a different network
than the browser. #1
- Windows doesn't work out of the box. The `scp` utility is required in PATH.
