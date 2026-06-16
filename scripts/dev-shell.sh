#!/usr/bin/env bash
set -euo pipefail

# Launch an interactive shell wired to the in-repo curlew and its hook, isolated
# from your real shell config. Lets you test hook changes before pushing without
# fighting an installed (e.g. Homebrew) curlew for PATH precedence.
#
# Usage: scripts/dev-shell.sh [zsh|bash]   (default: zsh)
#
# Inside the shell, `curlew` is bin/curlew from this repo and the hook is loaded,
# so `curl ... | bash` routes through your working copy. Exit to return to normal.

REPO="$(cd "$(dirname "$0")/.." && pwd)"
shell="${1:-zsh}"

case "$shell" in
  zsh)
    zdotdir="$(mktemp -d "${TMPDIR:-/tmp}/curlew-dev.XXXXXX")"
    trap 'rm -rf "$zdotdir"' EXIT
    cat > "$zdotdir/.zshrc" <<EOF
export PATH="$REPO/bin:\$PATH"
eval "\$(curlew --hook zsh)"
PROMPT='%F{yellow}curlew-dev%f %1~ %# '
print -P "%F{yellow}curlew dev shell%f — \$(curlew --version), zsh hook loaded. exit to leave."
EOF
    ZDOTDIR="$zdotdir" zsh -i
    ;;
  bash)
    rcfile="$(mktemp "${TMPDIR:-/tmp}/curlew-dev.XXXXXX")"
    trap 'rm -f "$rcfile"' EXIT
    cat > "$rcfile" <<EOF
export PATH="$REPO/bin:\$PATH"
eval "\$(curlew --hook bash)"
PS1='curlew-dev \w \$ '
echo "curlew dev shell — \$(curlew --version), bash hook loaded. exit to leave."
EOF
    bash --rcfile "$rcfile" -i
    ;;
  *)
    echo "usage: $(basename "$0") [zsh|bash]" >&2
    exit 1
    ;;
esac
