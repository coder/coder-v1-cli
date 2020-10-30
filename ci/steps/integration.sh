#!/bin/bash

set -eo pipefail

log() {
  echo "--- $@"
}

cd "$(git rev-parse --show-toplevel)"

log "building integration test image"
docker build -f ./ci/integration/Dockerfile -t coder-cli-integration:latest .

log "starting integration tests"
go test ./ci/integration -count=1
