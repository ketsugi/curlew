#!/usr/bin/env bash
set -euo pipefail

# Build the curlew binary.
# Usage: scripts/build-dist.sh [output-path]

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUTPUT="${1:-$REPO_ROOT/dist/curlew}"

export GOPROXY="${GOPROXY:-direct}"

mkdir -p "$(dirname "$OUTPUT")"
rm -f "$OUTPUT"

go build -o "$OUTPUT" "$REPO_ROOT/cmd/curlew/"

echo "Built: $OUTPUT"
