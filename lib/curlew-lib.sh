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
