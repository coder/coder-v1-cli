# Coder CLI

[![GitHub Release](https://img.shields.io/github/v/release/cdr/coder-cli?color=6b9ded&include_prerelease=false)](https://github.com/cdr/coder-cli/releases)
[![Documentation](https://godoc.org/cdr.dev/coder-cli?status.svg)](https://pkg.go.dev/cdr.dev/coder-cli/coder-sdk)

> **Note**: This is the command line utility for [Coder Classic](https://coder.com/docs/coder).
> If you are using [Coder OSS](https://coder.com/docs/coder-oss/latest), use [these instructions](https://coder.com/docs/coder-oss/latest/install)
> to install the CLI.

## Code

As of v1.24.0, the Coder CLI is closed source. The code in this repo will remain
as it was when closed on 20 October 2021. We will continue to use issues and
releases for the time being, but this may change.

We recommend using the SDK included in this repo until we publish the new Go SDK
that's currently in progress.

We will not accept any further pull requests.

## Bugs & feature requests

To report bugs and request features, please [open an issue](https://github.com/cdr/coder-cli/issues/new).

## Installation

### Homebrew (Mac, Linux)

```sh
brew install cdr/coder/coder-cli
```

### Linux
```sh

#download latest release
sudo wget https://github.com/coder/coder-cli/releases/download/{RELEASE-VERSION}/coder-cli-linux-amd64.tar.gz

#extract .tar.gz
sudo tar -xf ./coder-cli-linux-amd64.tar.gz

#add execution permissions
sudo chmod +x ./coder

#make coder binary globally available
sudo mv ./coder /usr/local/bin/coder

#cleanup
sudo rm coder-cli-linux-amd64.tar.gz
```

### Download (Windows, Linux, Mac)

Download the latest [release](https://github.com/cdr/coder-cli/releases):

1. Click a release and download the tar file for your operating system (ex: coder-cli-linux-amd64.tar.gz)
2. Extract the `coder` binary.

## Usage

View the usage documentation [here](./docs/coder.md).

You can find additional Coder usage documentation on [coder.com/docs/cli](https://coder.com/docs/cli).
