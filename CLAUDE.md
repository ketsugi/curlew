# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

curlew is a bash CLI that wraps `curl | bash`: it downloads a script, validates it's text (not a binary), offers visual inspection and optional Claude-powered analysis, then asks for explicit confirmation before executing. It also emits shell hooks that transparently intercept `curl|bash` pipelines.

## Commands

```bash
bats test/                      # run all tests
bats test/curlew-lib.bats       # run a single test file
bin/curlew <url-or-file>        # run locally from repo root (no install needed)
scripts/build-dist.sh dist/curlew   # build the single-file release artifact
```

Tests require bats-core (`brew install bats-core` or `apt install bats`). There is no compile/lint step in CI beyond `bats test/` plus a smoke test of the built dist artifact.

## Architecture

Two-file split, by testability:

- `bin/curlew` — entrypoint: argument parsing, the interactive flow (download → validate → inspect → analyze → confirm → execute), and the shell-hook emitters (`__curlew_emit_hook_zsh` / `__curlew_emit_hook_bash`). Side-effecting and TTY-bound, so it's exercised by integration tests rather than unit tests.
- `lib/curlew-lib.sh` — pure, sourceable functions with no side effects: `validate_mimetype`, `has_null_bytes`, `has_injection_patterns`, `validate_shebang`, `get_interpreter`. This is where unit-testable logic lives.

`bin/curlew` sources the lib via `source "${CURLEW_LIB:-$CURLEW_ROOT/lib/curlew-lib.sh}"`. The `CURLEW_LIB` override is how tests (and the dist build) point at the lib directly. When adding testable logic, put the function in the lib, call it from `bin/curlew`, and add a test in `test/curlew-lib.bats`.

### Dist build

`scripts/build-dist.sh` produces a zero-dependency single file by `sed`-splicing `lib/curlew-lib.sh` into `bin/curlew` at the `source` line. The lib stays separate in the repo for testing; only the release artifact is inlined. `dist/` is gitignored — it's a build output, never committed.

### Shell hooks

The hooks are heredoc'd shell code emitted by `curlew --hook {zsh,bash}` for `eval` into the user's shell. zsh uses a `preexec` hook; bash uses a `DEBUG` trap with `extdebug` (returning non-zero skips the command). Both detect `curl|wget ... | (ba)sh`, extract the URL, and re-route through curlew. Scope is deliberately pipe-to-shell only — process substitution, `eval`, and two-step download-then-run are out of scope. Bypass is `CURLEW_BYPASS=1`. Because hooks run in the user's live shell, they will always be shell code regardless of the main binary's language.

## Conventions and constraints

- `bin/curlew`, `lib/curlew-lib.sh`, and `scripts/*` all run under `set -euo pipefail`.
- Optional dependencies (`claude`, `glow`, `openssl`) must degrade gracefully — guard every use with `command -v`.
- AI analysis treats the script as untrusted input: the prompt instructs Claude to ignore embedded instructions, the content is fenced with a random sentinel, and `has_injection_patterns` blocks analysis unless `--force-analyze` is passed. Preserve these guards when touching the analysis path.
- Execution honors the script's shebang via `validate_shebang` (which rejects multi-arg/unsafe shebangs) and `get_interpreter`, invoking the interpreter directly rather than piping — this keeps it working on `noexec` /tmp.
- Test-only env vars (not public API): `CURLEW_LIB`, `CURLEW_SKIP_TTY_CHECK`, `CURLEW_CLAUDE_CMD`, `CURLEW_MODEL`.

## Workflow

- Direct pushes to `main` are blocked; all changes go through a PR, one branch per change. CI must be green to merge.
- Architecture decisions are recorded in `docs/decisions/` (ADRs). Notably, ADR-007 sets the trigger for rewriting curlew off bash into Go/Rust: doing any two of issues #4 (configurable AI harness), #7 (diff/cache mode), #8 (static analysis). Check the ADRs before re-litigating a settled design question.
- Releasing: bump `VERSION` in `bin/curlew`, commit, tag `vX.Y.Z`, push with `--tags`. The release workflow attaches `bin/curlew` and a SHA-256 checksum.
