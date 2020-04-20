# coder

`coder` provides a one-way, live file sync to your Coder Enterprise environment.

It is useful in cases where you want to use an unsupported IDE with your Coder
Environment.

## Login
```shell script
$ coder login https://my-coder-enterprise.com
```

## Setting up a Live Sync

Ensure that `rsync` is installed locally and in your environment.

``
$ coder sync ~/Projects/cdr/enterprise/. my-env:~/enterprise
``

## Caveats

- The `coder login` flow will not work when the CLI is ran from a different network
than the browser. [Issue](https://github.com/cdr/coder-cli/issues/1)

## Sync Architecture

We decided to use `rsync` because other solutions are extremely slow for the initial
sync.

Later, we may use `mutagen` for a two-way sync alternative when
it supports custom transports.

