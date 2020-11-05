#!/bin/bash

set -eo pipefail

cd "$(git rev-parse --show-toplevel)"

echo "--- building integration test image"
docker build -f ./ci/integration/Dockerfile -t coder-cli-integration:latest .

echo "--- starting integration tests"
go test ./ci/integration -count=1
