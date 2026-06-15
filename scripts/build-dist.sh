#!/usr/bin/env bash
set -euo pipefail

# Build a single-file curlew artifact by inlining lib into bin.
# Usage: scripts/build-dist.sh [output-path]

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUTPUT="${1:-$REPO_ROOT/dist/curlew}"

mkdir -p "$(dirname "$OUTPUT")"

{
  sed -n '1,/^source "\${CURLEW_LIB:-/{
    /^CURLEW_ROOT=/d
    /^source "\${CURLEW_LIB:-/d
    /^# shellcheck source=/d
    /^# Source library functions/d
    p
  }' "$REPO_ROOT/bin/curlew"
  echo ""
  echo "# --- Inlined library functions ---"
  tail -n +2 "$REPO_ROOT/lib/curlew-lib.sh"
  echo ""
  echo "# --- Main script ---"
  sed '1,/^source "\${CURLEW_LIB:-/d' "$REPO_ROOT/bin/curlew"
} > "$OUTPUT"

chmod +x "$OUTPUT"
echo "Built: $OUTPUT"
