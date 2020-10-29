# coder-cli

`coder` is a command line utility for Coder Enterprise.

To report bugs and request features, please [open an issue](https://github.com/cdr/coder-cli/issues/new).

## Usage

View the `coder-cli` documentation [here](./docs/coder.md).

You can find additional Coder Enterprise usage documentation on [https://enterprise.coder.com](https://enterprise.coder.com/docs/getting-started).

## Install Release

Download the latest [release](https://github.com/cdr/coder-cli/releases)

1. Click a release and download the tar file for your operating system (ex: coder-cli-linux-amd64.tar.gz)
2. Extract the `coder` binary from the tar file, ex:

```bash
cd ~/Downloads
tar -xvf ./coder-cli-darwin-amd64.tar.gz
./coder --help
```

Alternatively, use this helper script for MacOS and Linux

```bash
curl -fsSL https://raw.githubusercontent.com/cdr/coder-cli/master/install.sh | sh
```