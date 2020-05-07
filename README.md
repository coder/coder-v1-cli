# coder-cli

`coder` is a command line utility for Coder Enterprise.

## Install

```go
go get cdr.dev/coder-cli/cmd/coder
```

## Login
```shell script
$ coder login https://my-coder-enterprise.com
```

## Setting up a Live Sync

`coder sync` is useful in cases where you want to use an unsupported IDE with your Coder
Environment.

Ensure that `rsync` is installed locally and in your environment.

``
$ coder sync ~/Projects/cdr/enterprise/. my-env:~/enterprise
``

## Remote Terminal

You can access your environment's terminal with `coder sh <env>`. You can also
execute a command in your environment with `coder sh <env> [command] [args]`.

## Development URLs

You can retrieve the devurl of an environment.

``
$ coder url my-env 8080
``

## Caveats

- The `coder login` flow will not work when the CLI is ran from a different network
than the browser. [Issue](https://github.com/cdr/coder-cli/issues/1)

## Sync Architecture

We decided to use `rsync` because other solutions are extremely slow for the initial
sync.

Later, we may use `mutagen` for a two-way sync alternative when
it supports custom transports.

