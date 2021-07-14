#!/bin/bash

set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

if [[ $(git ls-files --other --modified --exclude-standard) ]]; then
  echo "Files have changed:"
  git ls-files --other --modified --exclude-standard
  git -c color.ui=never status
  exit 1
fi
