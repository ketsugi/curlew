# ADR-006: Shell hook implementation

**Status:** In progress (2026-06-15, stashed on `feat/shell-hook`)
**Issue:** [#11](https://github.com/ketsugi/curlew/issues/11) (open)

## Context

Intercept `curl | bash` commands transparently via shell hooks so users don't need to remember to type `curlew`.

## Design

- `curlew --hook zsh` outputs a zsh `preexec` function
- `curlew --hook bash` outputs a bash `DEBUG` trap (with `extdebug` for command suppression)
- Pattern: `(curl|wget)\s+.*\|\s*(sudo\s*)?(ba)?sh`
- Extracts URL via grep for `https?://` in the portion before the pipe
- `CURLEW_BYPASS=1` skips interception

## Activation

```bash
# .zshrc
eval "$(curlew --hook zsh)"

# .bashrc
eval "$(curlew --hook bash)"
```

## Implementation status

- Hook scripts: `lib/hooks/zsh.sh` and `lib/hooks/bash.sh` — done
- `--hook` flag in argument parsing — done
- Tests (15 passing) — done
- **Blocked on:** dist build inlining. The single-file artifact needs hooks embedded as heredocs. The sed-based build script struggles with this on macOS. Two options:
  1. Make `--hook` a Homebrew/repo-install-only feature (simplest)
  2. Rework the build script to handle heredoc embedding cleanly

## Stashed work

```bash
cd /Volumes/Workspaces/curlew
git stash list  # WIP on feat/shell-hook
git stash pop   # to resume
```

## Open questions

- Should the zsh hook use `kill -INT $$` to cancel the command, or is there a cleaner way?
- Fish support as a stretch goal?
