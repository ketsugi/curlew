#!/usr/bin/env bash
# curlew-lib.sh — testable functions extracted from curlew

# Validate that a file has a text-based MIME type.
# Returns 0 if valid, 1 if not. Prints the mimetype to stdout.
validate_mimetype() {
  local file="$1"
  local mimetype
  mimetype=$(file --brief --mime-type "$file")
  echo "$mimetype"
  case "$mimetype" in
    text/*|application/x-shellscript|application/javascript) return 0 ;;
    *) return 1 ;;
  esac
}

# Check if a file contains null bytes.
# Returns 0 if null bytes are found (i.e., likely binary), 1 if clean.
has_null_bytes() {
  local file="$1"
  if tr -d '\0' < "$file" | cmp -s - "$file"; then
    return 1  # no null bytes
  else
    return 0  # has null bytes
  fi
}

# Check if a file contains potential LLM prompt injection patterns.
# Returns 0 if injection patterns detected, 1 if clean.
has_injection_patterns() {
  local file="$1"
  grep -qiE '(ignore (all )?(previous|above|prior) instructions|you are now|disregard (the |all )?(above|previous|prior)|forget your (instructions|prompt)|new instructions:)' "$file" 2>/dev/null
}

# Parse and validate a shebang line.
# Returns 0 if shebang is safe to execute, 1 if not.
# On failure, prints the rejection reason to stderr.
validate_shebang() {
  local shebang="$1"

  if [[ ! "$shebang" =~ ^#!(.+) ]]; then
    # No shebang — will default to bash, which is fine
    return 0
  fi

  local interp_str="${BASH_REMATCH[1]}"
  local -a interp
  read -ra interp <<< "$interp_str"

  if [[ ${#interp[@]} -le 1 ]]; then
    # Single interpreter, no flags — always safe
    return 0
  fi

  case "${interp[0]##*/}" in
    env)
      local -a args=("${interp[@]:1}")
      [[ "${args[0]:-}" == "-S" ]] && args=("${args[@]:1}")
      if [[ ${#args[@]} -lt 1 ]]; then
        echo "Refusing degenerate env shebang: ${interp[*]}" >&2
        return 1
      fi
      if [[ ${#args[@]} -gt 1 ]]; then
        echo "Refusing complex env shebang: ${interp[*]}" >&2
        return 1
      fi
      ;;
    bash|sh|perl|python|python3|ruby|node)
      if [[ ${#interp[@]} -eq 2 ]]; then
        case "${interp[1]}" in
          -[wuexO]|-OO) ;;  # known benign flags
          *)
            echo "Refusing shebang flag: ${interp[1]}" >&2
            return 1
            ;;
        esac
      else
        echo "Refusing multi-arg shebang: ${interp[*]}" >&2
        return 1
      fi
      ;;
    *)
      echo "Refusing multi-arg shebang: ${interp[*]}" >&2
      return 1
      ;;
  esac

  return 0
}

# Resolve the AI analysis backend to a command string from environment config.
# Reads:
#   CURLEW_AI         backend preset: "claude" (default) or "ollama"
#   CURLEW_MODEL      model name (preset-specific; required for ollama)
#   CURLEW_AI_CMD     raw command override; wins over any preset
#   CURLEW_CLAUDE_CMD claude binary override (used by the claude preset)
# The resolved command is run with the analysis prompt on stdin and must write
# the analysis to stdout. Prints the command on success; on invalid config,
# prints the reason to stderr and returns non-zero.
resolve_ai_command() {
  local override="${CURLEW_AI_CMD:-}"
  if [[ -n "$override" ]]; then
    printf '%s\n' "$override"
    return 0
  fi

  local ai="${CURLEW_AI:-claude}"
  local model="${CURLEW_MODEL:-}"

  case "$ai" in
    claude)
      printf '%s --model %s --print\n' "${CURLEW_CLAUDE_CMD:-claude}" "${model:-sonnet}"
      ;;
    ollama)
      if [[ -z "$model" ]]; then
        echo "CURLEW_AI=ollama requires CURLEW_MODEL (e.g. CURLEW_MODEL=llama3.2)" >&2
        return 1
      fi
      printf 'ollama run %s\n' "$model"
      ;;
    *)
      echo "Unknown CURLEW_AI backend: $ai (supported: claude, ollama; or set CURLEW_AI_CMD)" >&2
      return 1
      ;;
  esac
}

# Get the interpreter command array for a given shebang.
# Prints the interpreter command to stdout (space-separated).
# If no shebang, prints "bash".
get_interpreter() {
  local shebang="$1"
  if [[ "$shebang" =~ ^#!(.+) ]]; then
    echo "${BASH_REMATCH[1]}"
  else
    echo "bash"
  fi
}
