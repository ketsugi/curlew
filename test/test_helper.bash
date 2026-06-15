#!/usr/bin/env bash
# test_helper.bash — loaded by each test file

CURLEW_ROOT="$(cd "$(dirname "$BATS_TEST_FILENAME")/.." && pwd)"
export CURLEW_LIB="$CURLEW_ROOT/lib/curlew-lib.sh"

# Source the library
source "$CURLEW_LIB"
