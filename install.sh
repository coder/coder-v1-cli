#!/bin/sh
# coder-cli installation helper script
# fork of https://github.com/denoland/deno_install
# TODO(everyone): Keep this script simple and easily auditable.

set -e

if [ "$(uname -m)" != "x86_64" ]; then
  echo "Error: Unsupported architecture $(uname -m). Only x64 binaries are available." 1>&2
  exit 1
fi

if [ "$OS" = "Windows_NT" ]; then
  target="windows-386"
  extension=".zip"
  if ! command -v unzip >/dev/null; then
    echo "Error: unzip is required to install coder-cli" 1>&2
    exit 1
  fi
else
  if ! command -v tar >/dev/null; then
    echo "Error: tar is required to install coder-cli" 1>&2
    exit 1
  fi
  extension=".tar.gz"
  case $(uname -s) in
  Darwin) target="darwin-amd64" ;;
  *) target="linux-amd64" ;;
  esac
fi

version=${1:-""}
if [ "$version" = "" ]; then
  coder_asset_path=$(
    curl -sSf https://github.com/cdr/coder-cli/releases |
      grep -o "/cdr/coder-cli/releases/download/.*/coder-cli-${target}${extension}" |
      head -n 1
  )
  if [ ! "$coder_asset_path" ]; then
    echo "Error: Unable to find latest coder-cli release on GitHub." 1>&2
    exit 1
  fi
  cdr_uri="https://github.com${coder_asset_path}"
else
  cdr_uri="https://github.com/cdr/coder-cli/releases/download/${1}/coder-cli-${target}${extension}"
fi

coder_install="${CODER_INSTALL:-$HOME/.coder}"
bin_dir="$coder_install/bin"
exe="$bin_dir/coder"

if [ ! -d "$bin_dir" ]; then
  mkdir -p "$bin_dir"
fi

curl --fail --location --progress-bar --output "$exe$extension" "$cdr_uri"
if [ "$extension" = ".zip" ]; then
  unzip -d "$bin_dir" -o "$exe$extension"
else
  tar -xzf "$exe$extension" -C "$bin_dir"
fi
chmod +x "$exe"
rm "$exe$extension"

echo "Coder was installed successfully to $exe"
if command -v coder >/dev/null; then
  echo "Run 'coder --help' to get started"
else
  case $SHELL in
  /bin/zsh) shell_profile=".zshrc" ;;
  *) shell_profile=".bash_profile" ;;
  esac
  echo "Manually add the directory to your \$HOME/$shell_profile (or similar)"
  echo "  export CODER_INSTALL=\"$coder_install\""
  echo "  export PATH=\"\$CODER_INSTALL/bin:\$PATH\""
  echo "Run '$exe --help' to get started"
fi
