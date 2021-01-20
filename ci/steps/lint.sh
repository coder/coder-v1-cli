#!/bin/bash

set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

echo "--- golangci-lint"
golangci-lint run -c .golangci.yml
