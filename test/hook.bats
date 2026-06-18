#!/usr/bin/env bats
# Tests for the shell hook feature (curlew --hook)

setup() {
  CURLEW_ROOT="$(cd "$(dirname "$BATS_TEST_FILENAME")/.." && pwd)"
  CURLEW="${CURLEW:-$CURLEW_ROOT/bin/curlew}"

  # Shared regex — must match the pattern used in the hook code.
  # If the hook regex changes, update it here.
  HOOK_REGEX='(^|[[:space:]])(curl|wget)[[:space:]]+[^|]+\|[[:space:]]*(ba)?sh([[:space:]]|$)'
}

# --- curlew --hook flag ---

@test "should output zsh preexec hook when --hook zsh" {
  run "$CURLEW" --hook zsh
  [ "$status" -eq 0 ]
  [[ "$output" == *"__curlew_preexec"* ]]
  [[ "$output" == *"add-zsh-hook"* ]]
}

@test "should output bash debug trap when --hook bash" {
  run "$CURLEW" --hook bash
  [ "$status" -eq 0 ]
  [[ "$output" == *"__curlew_trap_debug"* ]]
  [[ "$output" == *"extdebug"* ]]
}

@test "should fail when --hook given no argument" {
  run "$CURLEW" --hook
  [ "$status" -ne 0 ]
  [[ "$output" == *"argument"* ]]
}

@test "should fail when --hook given unsupported shell" {
  run "$CURLEW" --hook fish
  [ "$status" -ne 0 ]
  [[ "$output" == *"Unsupported shell"* ]]
}

# --- Hook evaluates cleanly in its target shell ---

@test "should parse cleanly when zsh hook is evaluated in zsh" {
  command -v zsh >/dev/null || skip "zsh not installed"
  run zsh -c "eval \"\$('$CURLEW' --hook zsh)\""
  [ "$status" -eq 0 ]
}

@test "should parse cleanly when bash hook is evaluated in bash" {
  run bash -c "eval \"\$('$CURLEW' --hook bash)\""
  [ "$status" -eq 0 ]
}

# --- zsh hook interception (fires the real preexec function) ---
# Stub curlew with a marker and stub kill so the test shell is not signalled.

@test "should intercept and route curl|bash when zsh hook fires" {
  command -v zsh >/dev/null || skip "zsh not installed"
  run zsh -c "
    eval \"\$('$CURLEW' --hook zsh)\"
    curlew() { print -r -- \"STUB_HIT:\$*\"; }
    kill() { :; }
    __curlew_preexec 'curl -fsSL https://example.com/install.sh | bash'
  "
  [[ "$output" == *"STUB_HIT:https://example.com/install.sh"* ]]
}

@test "should not intercept when sudo in pipe target (zsh runtime)" {
  command -v zsh >/dev/null || skip "zsh not installed"
  run zsh -c "
    eval \"\$('$CURLEW' --hook zsh)\"
    curlew() { print -r -- STUB_HIT; }
    kill() { :; }
    __curlew_preexec 'curl -fsSL https://example.com | sudo bash'
  "
  [[ "$output" != *STUB_HIT* ]]
}

@test "should honor inline CURLEW_BYPASS=1 prefix (zsh runtime)" {
  command -v zsh >/dev/null || skip "zsh not installed"
  run zsh -c "
    eval \"\$('$CURLEW' --hook zsh)\"
    curlew() { print -r -- STUB_HIT; }
    kill() { :; }
    __curlew_preexec 'CURLEW_BYPASS=1 curl -fsSL https://example.com/install.sh | bash'
  "
  [[ "$output" != *STUB_HIT* ]]
}

# --- Bypass detection ---

@test "should include CURLEW_BYPASS in zsh hook output" {
  run "$CURLEW" --hook zsh
  [[ "$output" == *'CURLEW_BYPASS'* ]]
}

@test "should include CURLEW_BYPASS in bash hook output" {
  run "$CURLEW" --hook bash
  [[ "$output" == *'CURLEW_BYPASS'* ]]
}

@test "should detect bypass when CURLEW_BYPASS=1 is first token" {
  cmd='CURLEW_BYPASS=1 curl -fsSL https://example.com/install.sh | bash'
  [[ "$cmd" =~ (^|[[:space:]])CURLEW_BYPASS=1[[:space:]] ]]
}

@test "should detect bypass when CURLEW_BYPASS=1 is non-first env var" {
  cmd='DEBUG=1 CURLEW_BYPASS=1 curl https://example.com | bash'
  [[ "$cmd" =~ (^|[[:space:]])CURLEW_BYPASS=1[[:space:]] ]]
}

# --- Pattern matching (positive cases) ---

@test "should match when simple curl piped to bash" {
  cmd='curl -fsSL https://example.com/install.sh | bash'
  [[ "$cmd" =~ $HOOK_REGEX ]]
}

@test "should match when wget piped to sh" {
  cmd='wget -O - https://example.com/setup.sh | sh'
  [[ "$cmd" =~ $HOOK_REGEX ]]
}

@test "should match when curl has many flags piped to sh" {
  cmd='curl --proto =https --tlsv1.2 -sSf https://sh.rustup.rs | sh'
  [[ "$cmd" =~ $HOOK_REGEX ]]
}

@test "should match when bash has trailing args" {
  cmd='curl https://example.com/install.sh | bash -s -- arg'
  [[ "$cmd" =~ $HOOK_REGEX ]]
}

# --- Pattern matching (negative cases) ---

@test "should not match when curl pipes to grep" {
  cmd='curl -fsSL https://example.com/data.json | grep foo'
  ! [[ "$cmd" =~ $HOOK_REGEX ]]
}

@test "should not match when curl has no pipe" {
  cmd='curl -fsSL https://example.com/file.tar.gz -o file.tar.gz'
  ! [[ "$cmd" =~ $HOOK_REGEX ]]
}

@test "should not match when multiple pipes present (curl|tee|bash)" {
  cmd='curl https://example.com | tee /tmp/log | bash'
  ! [[ "$cmd" =~ $HOOK_REGEX ]] || false
}

@test "should not match when command is xcurl (substring of curl)" {
  cmd='xcurl https://example.com/install.sh | bash'
  ! [[ "$cmd" =~ $HOOK_REGEX ]]
}

@test "should not match when pipe target is bashfoo (trailing chars)" {
  cmd='curl https://example.com/install.sh | bashfoo'
  ! [[ "$cmd" =~ $HOOK_REGEX ]]
}

@test "should not match when pipe target is shazam (trailing chars on sh)" {
  cmd='curl https://example.com/install.sh | shazam'
  ! [[ "$cmd" =~ $HOOK_REGEX ]]
}

# --- Sudo handling ---

@test "should skip interception when sudo in pipe target" {
  cmd='curl -fsSL https://example.com/install.sh | sudo bash'
  [[ "$cmd" =~ \|[[:space:]]*sudo ]]
}

# --- URL extraction ---

@test "should extract URL from curl command" {
  dl_cmd='curl -fsSL https://example.com/install.sh '
  url=$(printf '%s' "$dl_cmd" | grep -oE 'https?://[^[:space:]"'"'"')>;]+' | head -1)
  [ "$url" = "https://example.com/install.sh" ]
}

@test "should extract URL from wget command" {
  dl_cmd='wget -O - https://get.sdkman.io '
  url=$(printf '%s' "$dl_cmd" | grep -oE 'https?://[^[:space:]"'"'"')>;]+' | head -1)
  [ "$url" = "https://get.sdkman.io" ]
}

@test "should exclude semicolons from extracted URL" {
  dl_cmd='curl https://example.com/install.sh; echo done'
  url=$(printf '%s' "$dl_cmd" | grep -oE 'https?://[^[:space:]"'"'"')>;]+' | head -1)
  [ "$url" = "https://example.com/install.sh" ]
}
