#!/bin/bash

set -euo pipefail

echo "-- Generating docs"

cd "$(dirname "$0")"
cd ../../

rm -rf ./docs
mkdir ./docs
go run ./cmd/coder gen-docs ./docs

if ! command -v deno >/dev/null; then
  "deno is required to compile the docs into a single file"
  exit 1
fi

echo "-- Aggregating docs"
deno run --allow-read --allow-write ./ci/scripts/aggregate_docs.ts

if [[ ${CI-} && $(git ls-files --other --modified --exclude-standard) ]]; then
  echo "Documentation needs generation:"
  git -c color.ui=always status | grep --color=no '\e\[31m'
  echo "Please run the following locally:"
  echo "  ./ci/steps/gendocs.sh"
  exit 1
fi
