#!/bin/bash

# Make pushd and popd silent
pushd() { builtin pushd "$@" >/dev/null; }
popd() { builtin popd >/dev/null; }

set -euo pipefail

cd "$(git rev-parse --show-toplevel)/ci/steps"

tag=$(git describe --tags)

build() {
	echo "--- building coder-cli for $GOOS-$GOARCH"

	tmpdir=$(mktemp -d)
	go build -ldflags "-X cdr.dev/coder-cli/internal/version.Version=${tag}" -o "$tmpdir/coder" ../../cmd/coder
	# For MacOS builds to be notarized.
	cp ../gon.json $tmpdir/gon.json

	pushd "$tmpdir"
	case "$GOOS" in
		"windows")
			artifact="coder-cli-$GOOS-$GOARCH.zip"
			mv coder coder.exe
			zip "$artifact" coder.exe
			;;
		"linux")
			artifact="coder-cli-$GOOS-$GOARCH.tar.gz"
			tar -czf "$artifact" coder	
			;;
		"darwin")
		if [[ ${CI-} ]]; then
			artifact="coder-cli-$GOOS-$GOARCH.zip"
			gon -log-level debug ./gon.json
			mv coder.zip $artifact
		else
			artifact="coder-cli-$GOOS-$GOARCH.tar.gz"
			tar -czf "$artifact" coder	
			echo "--- warning: not in ci, skipping signed release of darwin"
		fi
			;;
	esac
	popd

	mkdir -p ../bin
	cp "$tmpdir/$artifact" ../bin/$artifact
	rm -rf "$tmpdir"
}

# Darwin builds do not work from Linux, so only try to build them from Darwin.
# See: https://github.com/cdr/coder-cli/issues/20
if [[ "$(uname)" == "Darwin" ]]; then
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 build
else
	echo "--- warning: Darwin builds don't work on Linux."
	echo "--- please use an OSX machine to build Darwin tars."
fi

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 build
CGO_ENABLED=0 GOOS=windows GOARCH=386 build
