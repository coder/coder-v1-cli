#!/bin/bash

set -euo pipefail

echo "Generating docs..."

cd "$(dirname "$0")"
cd ../../

rm -rf ./docs
mkdir ./docs
go run ./cmd/coder gen-docs ./docs

if [[ ${CI-} && $(git ls-files --other --modified --exclude-standard) ]]; then
  echo "Documentation needs generation:"
  git -c color.ui=always status | grep --color=no '\e\[31m'
  echo "Please run the following locally:"
  echo "  ./ci/steps/gendocs.sh"
  exit 1
fi
