#!/bin/bash

set -euo pipefail

echo "Linting..."

go vet ./...
golint -set_exit_status ./...
