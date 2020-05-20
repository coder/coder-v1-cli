#!/bin/bash

set -euo pipefail
cd "$(dirname "$0")"

export GOARCH=amd64
tag=$(git describe --tags)

mkdir -p bin

build(){
	tmpdir=$(mktemp -d)
	go build -ldflags "-X main.version=${tag}" -o "$tmpdir/coder-cli"

	pushd "$tmpdir"
		tarname="coder-cli-$GOOS-$GOARCH.tar.gz"
		tar -czf "$tarname" coder-cli
	popd

	cp "$tmpdir/$tarname" bin
	rm -rf "$tmpdir"
}

GOOS=darwin build
GOOS=linux build
