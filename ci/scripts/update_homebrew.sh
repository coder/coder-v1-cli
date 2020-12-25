#!/bin/bash

set -e

asset="$1"
sha="$(sha256sum "$asset" | awk '{ print $1 }')"
tag="$(git describe --tags)"

tmpdir=$(mktemp -d)

pushd "$tmpdir"
git clone https://github.com/cdr/homebrew-coder

pushd homebrew-coder

branch="coder-cli-release-$tag"
git checkout -b "$branch"

new_formula="$(cat <<EOF
class Coder < Formula
  desc "A command-line tool for the Coder remote development platform"
  homepage "https://github.com/cdr/coder-cli"
  url "https://github.com/cdr/coder-cli/releases/download/$tag/coder-cli-darwin-amd64.zip"
  version "$tag"
  sha256 "$sha"
  bottle :unneeded
  def install
    bin.install "coder"
  end
  test do
    system "#{bin}/coder", "--version"
  end
end
EOF
)"

echo "$new_formula" > coder.rb

git add coder.rb
git commit -m "chore: update Coder CLI to $tag"
git push --set-upstream origin "$branch"

gh pr create --fill

rm -rf "$tmpdir"
