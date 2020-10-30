#!/bin/sh
# forked from https://github.com/denoland/deno_install/blob/master/install_test.sh

set -e

cd "$(dirname "$0")"
cd ../../

# Test that we can install the latest version at the default location.
rm -f ~/.coder/bin/coder
unset CODER_INSTALL
sh ./install.sh
~/.coder/bin/coder --version

# Test that we can install a specific version at a custom location.
rm -rf ~/coder-1.12.2
export CODER_INSTALL="$HOME/coder-1.12.2"
./install.sh v1.12.2
~/coder-1.12.2/bin/coder --version | grep 1.12.2

# Test that we can install at a relative custom location.
export CODER_INSTALL="."
./install.sh v1.12.2
bin/coder --version | grep 1.12.2
rm ./bin/coder
