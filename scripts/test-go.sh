#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

echo "==> Running all tests..."
go test ./... -count=1

echo "==> All tests passed."
