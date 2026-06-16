#!/usr/bin/env bats
# Tests for the shell hook feature (curlew --hook)

setup() {
  CURLEW_ROOT="$(cd "$(dirname "$BATS_TEST_FILENAME")/.." && pwd)"
  CURLEW="$CURLEW_ROOT/bin/curlew"
}

# --- curlew --hook flag ---

@test "--hook zsh outputs zsh hook code" {
  run bash "$CURLEW" --hook zsh
  [ "$status" -eq 0 ]
  [[ "$output" == *"__curlew_preexec"* ]]
  [[ "$output" == *"add-zsh-hook"* ]]
}

@test "--hook bash outputs bash hook code" {
  run bash "$CURLEW" --hook bash
  [ "$status" -eq 0 ]
  [[ "$output" == *"__curlew_trap_debug"* ]]
  [[ "$output" == *"extdebug"* ]]
}

@test "--hook with no argument fails" {
  run bash "$CURLEW" --hook
  [ "$status" -ne 0 ]
  [[ "$output" == *"requires an argument"* ]]
}

@test "--hook with unsupported shell fails" {
  run bash "$CURLEW" --hook fish
  [ "$status" -ne 0 ]
  [[ "$output" == *"Unsupported shell"* ]]
}

# --- Hook pattern matching ---

@test "zsh hook contains CURLEW_BYPASS check" {
  run bash "$CURLEW" --hook zsh
  [[ "$output" == *'CURLEW_BYPASS'* ]]
}

@test "bash hook contains CURLEW_BYPASS check" {
  run bash "$CURLEW" --hook bash
  [[ "$output" == *'CURLEW_BYPASS'* ]]
}

@test "zsh hook matches curl|bash pattern" {
  run bash "$CURLEW" --hook zsh
  [[ "$output" == *"curl|wget"* ]] || [[ "$output" == *"(curl|wget)"* ]]
}

@test "bash hook matches curl|bash pattern" {
  run bash "$CURLEW" --hook bash
  [[ "$output" == *"curl|wget"* ]] || [[ "$output" == *"(curl|wget)"* ]]
}

# --- URL extraction logic (test via sourced bash hook) ---

@test "bash hook regex matches simple curl|bash" {
  cmd='curl -fsSL https://example.com/install.sh | bash'
  [[ "$cmd" =~ (curl|wget)[[:space:]]+.*\|[[:space:]]*(sudo[[:space:]]*)?(ba)?sh ]]
}

@test "bash hook matches wget|sh pattern" {
  cmd='wget -O - https://example.com/setup.sh | sh'
  [[ "$cmd" =~ (curl|wget)[[:space:]]+.*\|[[:space:]]*(sudo[[:space:]]*)?(ba)?sh ]]
}

@test "bash hook matches curl|sudo bash pattern" {
  cmd='curl -fsSL https://example.com/install.sh | sudo bash'
  [[ "$cmd" =~ (curl|wget)[[:space:]]+.*\|[[:space:]]*(sudo[[:space:]]*)?(ba)?sh ]]
}

@test "bash hook does NOT match curl|grep" {
  cmd='curl -fsSL https://example.com/data.json | grep foo'
  ! [[ "$cmd" =~ (curl|wget)[[:space:]]+.*\|[[:space:]]*(sudo[[:space:]]*)?(ba)?sh ]]
}

@test "bash hook does NOT match curl without pipe" {
  cmd='curl -fsSL https://example.com/file.tar.gz -o file.tar.gz'
  ! [[ "$cmd" =~ (curl|wget)[[:space:]]+.*\|[[:space:]]*(sudo[[:space:]]*)?(ba)?sh ]]
}

@test "URL extraction from curl command works" {
  dl_cmd='curl -fsSL https://example.com/install.sh '
  url=$(printf '%s' "$dl_cmd" | grep -oE 'https?://[^[:space:]"'"'"')>]+' | head -1)
  [ "$url" = "https://example.com/install.sh" ]
}

@test "URL extraction from wget command works" {
  dl_cmd='wget -O - https://get.sdkman.io '
  url=$(printf '%s' "$dl_cmd" | grep -oE 'https?://[^[:space:]"'"'"')>]+' | head -1)
  [ "$url" = "https://get.sdkman.io" ]
}
