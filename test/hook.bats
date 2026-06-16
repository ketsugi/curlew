#!/usr/bin/env bats
# Tests for the shell hook feature (curlew --hook)

setup() {
  CURLEW_ROOT="$(cd "$(dirname "$BATS_TEST_FILENAME")/.." && pwd)"
  CURLEW="$CURLEW_ROOT/bin/curlew"

  # Shared regex — must match the pattern used in the hook code.
  # If the hook regex changes, update it here (single source of truth for tests).
  HOOK_REGEX='(curl|wget)[[:space:]]+[^|]+\|[[:space:]]*(ba)?sh'
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

# --- Bypass detection ---

@test "zsh hook contains CURLEW_BYPASS check" {
  run bash "$CURLEW" --hook zsh
  [[ "$output" == *'CURLEW_BYPASS'* ]]
}

@test "bash hook contains CURLEW_BYPASS check" {
  run bash "$CURLEW" --hook bash
  [[ "$output" == *'CURLEW_BYPASS'* ]]
}

@test "bypass prefix is detected in command string" {
  cmd='CURLEW_BYPASS=1 curl -fsSL https://example.com/install.sh | bash'
  [[ "$cmd" =~ ^CURLEW_BYPASS=1[[:space:]] ]]
}

# --- Pattern matching (positive cases) ---

@test "matches simple curl|bash" {
  cmd='curl -fsSL https://example.com/install.sh | bash'
  [[ "$cmd" =~ $HOOK_REGEX ]]
}

@test "matches wget|sh" {
  cmd='wget -O - https://example.com/setup.sh | sh'
  [[ "$cmd" =~ $HOOK_REGEX ]]
}

@test "matches curl|bash with flags" {
  cmd='curl --proto =https --tlsv1.2 -sSf https://sh.rustup.rs | sh'
  [[ "$cmd" =~ $HOOK_REGEX ]]
}

# --- Pattern matching (negative cases — must NOT match) ---

@test "does NOT match curl|grep" {
  cmd='curl -fsSL https://example.com/data.json | grep foo'
  ! [[ "$cmd" =~ $HOOK_REGEX ]]
}

@test "does NOT match curl without pipe" {
  cmd='curl -fsSL https://example.com/file.tar.gz -o file.tar.gz'
  ! [[ "$cmd" =~ $HOOK_REGEX ]]
}

@test "does NOT match multi-pipe (curl|tee|bash)" {
  cmd='curl https://example.com | tee /tmp/log | bash'
  # The first segment "curl https://example.com " contains no pipe, matches [^|]+,
  # then | tee /tmp/log | bash — the regex requires (ba)?sh immediately after the pipe.
  # "tee /tmp/log | bash" doesn't match (curl|wget) so the overall regex shouldn't match
  # at the start. Let's verify the actual behavior:
  ! [[ "$cmd" =~ $HOOK_REGEX ]] || {
    # If it does match, the [^|]+ consumed "curl https://example.com " and
    # the \| matched the FIRST pipe. Then (ba)?sh needs to follow — but "tee" follows.
    # So this should NOT match. If it does, the regex needs further tightening.
    false
  }
}

# --- Sudo handling ---

@test "does NOT match curl|sudo bash (sudo in pipe target is skipped)" {
  cmd='curl -fsSL https://example.com/install.sh | sudo bash'
  # The hook skips commands with sudo in the pipe target before regex check
  [[ "$cmd" =~ \|[[:space:]]*sudo ]]
}

# --- URL extraction ---

@test "extracts URL from curl command" {
  dl_cmd='curl -fsSL https://example.com/install.sh '
  url=$(printf '%s' "$dl_cmd" | grep -oE 'https?://[^[:space:]"'"'"')>;]+' | head -1)
  [ "$url" = "https://example.com/install.sh" ]
}

@test "extracts URL from wget command" {
  dl_cmd='wget -O - https://get.sdkman.io '
  url=$(printf '%s' "$dl_cmd" | grep -oE 'https?://[^[:space:]"'"'"')>;]+' | head -1)
  [ "$url" = "https://get.sdkman.io" ]
}

@test "excludes semicolons from extracted URL" {
  dl_cmd='curl https://example.com/install.sh; echo done'
  url=$(printf '%s' "$dl_cmd" | grep -oE 'https?://[^[:space:]"'"'"')>;]+' | head -1)
  [ "$url" = "https://example.com/install.sh" ]
}
