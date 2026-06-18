package hook

// ZshHook returns the zsh preexec hook code for eval.
func ZshHook() string {
	return zshHook
}

// BashHook returns the bash DEBUG trap hook code for eval.
func BashHook() string {
	return bashHook
}

const zshHook = `# curlew shell hook for zsh
# Intercepts curl|bash commands and routes them through curlew.
# Scope: pipe-to-shell only (not process-sub, eval, or two-step forms).
# Bypass with: CURLEW_BYPASS=1 curl ... | bash

__curlew_preexec() {
  # Allow explicit bypass via env var.
  [[ "${CURLEW_BYPASS:-}" == "1" ]] && return

  local cmd="$1"

  # All match patterns are held in variables. zsh mishandles | in an inline
  # [[ =~ ]] regex two ways: an unquoted | is a parse error, and a
  # backslash-escaped \| is stripped to a bare | — an empty alternation that
  # matches everything (so an inline sudo check would skip every command).
  # Expanding the pattern from a variable sidesteps both.

  # Allow explicit bypass via inline prefix (CURLEW_BYPASS=1 in leading assignments).
  local bypass_re='(^|[[:space:]])CURLEW_BYPASS=1[[:space:]]'
  [[ "$cmd" =~ $bypass_re ]] && return

  # Skip commands with sudo in the pipe target — let users explicitly run sudo curlew.
  local sudo_re='\|[[:space:]]*sudo'
  [[ "$cmd" =~ $sudo_re ]] && return

  # Detect: curl/wget ... | bash/sh (single pipe only, anchored to avoid substring matches)
  local re='(^|[[:space:]])(curl|wget)[[:space:]]+[^|]+\|[[:space:]]*(ba)?sh([[:space:]]|$)'
  if [[ "$cmd" =~ $re ]]; then
    # Extract URL from the download command (portion before the pipe)
    local dl_cmd="${cmd%%|*}"
    local url
    url=$(printf '%s' "$dl_cmd" | grep -oE 'https?://[^[:space:]"'"'"')>;]+' | head -1)

    if [[ -n "$url" ]]; then
      printf '\033[1;33m⚠ curlew:\033[0m Intercepted pipe-to-shell. Routing through curlew...\n'
      curlew "$url"
      kill -INT $$
    fi
  fi
}

autoload -Uz add-zsh-hook
add-zsh-hook preexec __curlew_preexec
`

const bashHook = `# curlew shell hook for bash
# Intercepts curl|bash commands and routes them through curlew.
# Scope: pipe-to-shell only (not process-sub, eval, or two-step forms).
# Bypass with: CURLEW_BYPASS=1 curl ... | bash
# Requires: bash 4.4+ (for extdebug trap return value support)

__curlew_trap_debug() {
  # Allow explicit bypass (env var or inline prefix anywhere in leading assignments)
  [[ "${CURLEW_BYPASS:-}" == "1" ]] && return 0
  [[ "$BASH_COMMAND" =~ (^|[[:space:]])CURLEW_BYPASS=1[[:space:]] ]] && return 0

  local cmd="$BASH_COMMAND"

  # Only intercept top-level commands (skip subshells and function internals)
  [[ "$BASH_SUBSHELL" -gt 0 ]] && return 0

  # Skip commands with sudo in the pipe target — let users explicitly run sudo curlew
  [[ "$cmd" =~ \|[[:space:]]*sudo ]] && return 0

  # Detect: curl/wget ... | bash/sh (single pipe only, anchored to avoid substring matches)
  if [[ "$cmd" =~ (^|[[:space:]])(curl|wget)[[:space:]]+[^|]+\|[[:space:]]*(ba)?sh([[:space:]]|$) ]]; then
    # Extract URL from the download command (portion before the pipe)
    local dl_cmd="${cmd%%|*}"
    local url
    url=$(printf '%s' "$dl_cmd" | grep -oE 'https?://[^[:space:]"'"'"')>;]+' | head -1)

    if [[ -n "$url" ]]; then
      printf '\033[1;33m⚠ curlew:\033[0m Intercepted pipe-to-shell. Routing through curlew...\n'
      curlew "$url"
      # Return non-zero to prevent the original command from executing
      return 1
    fi
  fi

  return 0
}

# extdebug: when the DEBUG trap returns non-zero, the command is skipped
shopt -s extdebug
trap '__curlew_trap_debug' DEBUG
`
