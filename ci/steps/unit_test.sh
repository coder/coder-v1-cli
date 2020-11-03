#!/bin/bash

set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

echo "--- go test..."

go test $(go list ./... | grep -v pkg/tcli | grep -v ci/integration | grep -v coder-sdk)
