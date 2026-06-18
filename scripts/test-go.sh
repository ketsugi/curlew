#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

echo "==> Building binary..."
go build -o bin/curlew-go ./cmd/curlew/

echo "==> Running Go unit tests..."
go test ./... -count=1

echo "==> Running bats integration tests against Go binary..."
CURLEW="$(pwd)/bin/curlew-go" bats test/curlew-integration.bats test/hook.bats

echo "==> All tests passed."
