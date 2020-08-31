#!/bin/bash

# Make pushd and popd silent
pushd() { builtin pushd "$@" >/dev/null; }
popd() { builtin popd >/dev/null; }

set -euo pipefail
cd "$(dirname "$0")"

export GOARCH=amd64
tag=$(git describe --tags)

build() {
	tmpdir=$(mktemp -d)
	go build -ldflags "-X main.version=${tag}" -o "$tmpdir/coder" ../../cmd/coder

	pushd "$tmpdir"
	if [[ "$GOOS" == "windows" ]]; then
		artifact="coder-cli-$GOOS-$GOARCH.zip"
		mv coder coder.exe
		zip "$artifact" coder.exe
	else
		artifact="coder-cli-$GOOS-$GOARCH.tar.gz"
		tar -czf "$artifact" coder
	fi
	popd

	mkdir -p ../bin
	cp "$tmpdir/$artifact" ../bin/$artifact
	rm -rf "$tmpdir"
}

# Darwin builds do not work from Linux, so only try to build them from Darwin.
# See: https://github.com/cdr/coder-cli/issues/20
if [[ "$(uname)" == "Darwin" ]]; then
	GOOS=linux build
	CGO_ENABLED=1 GOOS=darwin build
	GOOS=windows GOARCH=386 build
	exit 0
fi

echo "Warning: Darwin builds don't work on Linux."
echo "Please use an OSX machine to build Darwin tars."
GOOS=linux build
