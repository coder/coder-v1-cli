#!/bin/bash

set -euo pipefail

cd "$(dirname "$0")"
cd ../../

echo "--- golangci-lint"
golangci-lint run -c .golangci.yml
