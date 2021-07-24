#!/bin/bash

# Make pushd and popd silent
pushd() { builtin pushd "$@" >/dev/null; }
popd() { builtin popd >/dev/null; }

set -euo pipefail

cd "$(git rev-parse --show-toplevel)/ci/scripts"

tag="$(git describe --tags)"

flavor="$GOOS"
if [[ "$GOOS" == "windows" ]]; then
	unset GOARCH
else
	flavor+="-$GOARCH"
fi
echo "--- building coder-cli for $flavor"

tmpdir="$(mktemp -d)"
go build -ldflags "-X cdr.dev/coder-cli/internal/version.Version=${tag}" -o "$tmpdir/coder" ../../cmd/coder

cp ../gon.json $tmpdir/gon.json

pushd "$tmpdir"
case "$GOOS" in
"windows")
	artifact="coder-cli-$GOOS.zip"
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
