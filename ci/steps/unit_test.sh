#!/bin/bash

set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

echo "--- running unit tests"
go test $(go list ./... | grep -v pkg/tcli | grep -v ci/integration | grep -v coder-sdk)
