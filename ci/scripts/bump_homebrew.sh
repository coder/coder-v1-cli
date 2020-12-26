#!/bin/bash

set -e

macos_asset="$1"
linux_asset="$2"

sha() {
  sha256sum "$1" | awk '{ print $1 }'
}
tag="$(git describe --tags)"
trimmed_tag="${tag#"v"}"

tmpdir=$(mktemp -d)

pushd "$tmpdir"
git clone https://github.com/cdr/homebrew-coder

pushd homebrew-coder

branch="bump-coder-cli-$tag"
git checkout -b "$branch"

if [[ "$GITHUB_TOKEN" != "" ]]; then
  git remote set-url origin "https://x-access-token:$GITHUB_TOKEN@github.com/cdr/homebrew-coder"
fi

new_formula="$(cat <<EOF
class CoderCli < Formula
  desc "Command-line tool for the Coder remote development platform"
  homepage "https://github.com/cdr/coder-cli"
  version "$trimmed_tag"
  bottle :unneeded

  if OS.mac?
    url "https://github.com/cdr/coder-cli/releases/download/$tag/coder-cli-darwin-amd64.zip"
    sha256 "$(sha "$macos_asset")"
  else
    url "https://github.com/cdr/coder-cli/releases/download/$tag/coder-cli-linux-amd64.tar.gz"
    sha256 "$(sha "$linux_asset")"
  end

  def install
    bin.install "coder"
  end
  test do
    system "#{bin}/coder", "--version"
  end
end
EOF
)"

echo "$new_formula" > coder-cli.rb

git diff
git add coder-cli.rb
git commit -m "chore: bump coder-cli to $tag"
git push --set-upstream origin "$branch"

gh pr create --fill

rm -rf "$tmpdir"
