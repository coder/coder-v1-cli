#!/bin/bash

set -euo pipefail

echo "Linting..."

cd "$(dirname "$0")"
cd ../../

echo "--- golangci-lint"
golangci-lint run -c .golangci.yml
