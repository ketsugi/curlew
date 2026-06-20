# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

curlew is a Go CLI that wraps `curl | bash`: it downloads a script, validates it's text (not a binary), offers visual inspection and optional Claude-powered analysis, then asks for explicit confirmation before executing. It also emits shell hooks that transparently intercept `curl|bash` pipelines.

## Commands

```bash
go test ./...                   # all tests (unit + e2e integration)
go build -o bin/curlew-go ./cmd/curlew/  # build locally
scripts/build-dist.sh           # build dist/curlew release artifact
```

All tests run via `go test` — no external test framework required.

## Architecture

```
cmd/curlew/main.go              — entrypoint (cobra CLI: root + list/setup subcommands)
internal/hook/                  — shell hook string constants
internal/validate/              — pure validation functions (MIME, null bytes,
                                  injection patterns, shebang, interpreter)
internal/ai/                    — AI backend resolution from config
internal/config/                — TOML config loading (XDG, env override)
internal/setup/                 — shell detection + idempotent hook install (curlew setup)
internal/ledger/                — install ledger: storage, lookup, analysis cache
internal/homograph/             — URL hostname homograph/punycode detection
internal/staticanalysis/        — AST-based structural analysis (mvdan.cc/sh)
internal/run/                   — interactive flow orchestration + terminal helpers
e2e/                            — integration tests (builds + execs the binary)
```

### Packages

- `internal/validate` — side-effect-free functions: `MIMEType`, `HasNullBytes`, `HasInjectionPatterns`, `ValidateShebang`, `GetInterpreter`. Unit-testable logic lives here.
- `internal/ai` — `ResolveCommand` resolves the AI backend from `CURLEW_AI` / `CURLEW_MODEL` / `CURLEW_AI_CMD` env vars into an argv slice.
- `internal/hook` — `ZshHook()` and `BashHook()` return the shell code emitted by `curlew --hook`.
- `internal/staticanalysis` — `Analyze([]byte)` walks the `mvdan.cc/sh` AST and returns categorized structural findings (network, file writes, package installs, privilege escalation, persistence, dangerous ops, obfuscation, URLs). Deterministic and dependency-light; the layer beneath the AI's semantic analysis. Falls back to silent skip on parse failure (non-shell scripts).
- `internal/ledger` — persistent record of vetted scripts (`Record`/`MarkExecuted`/`Lookup`/`List`), keyed by normalized-URL hash, with a per-entry AI analysis cache invalidated on script-hash change.
- `internal/run` — the interactive flow (download → validate → change-check → static analysis → inspect → AI analyze → confirm → execute) and terminal I/O.

### Shell hooks

The hooks are shell code emitted by `curlew --hook {zsh,bash}` for `eval` into the user's shell. zsh uses a `preexec` hook; bash uses a `DEBUG` trap with `extdebug` (returning non-zero skips the command). Both detect `curl|wget ... | (ba)sh`, extract the URL, and re-route through curlew. Scope is deliberately pipe-to-shell only — process substitution, `eval`, and two-step download-then-run are out of scope. Bypass is `CURLEW_BYPASS=1`. The hooks are always shell code regardless of the binary's language.

### Dist build

`scripts/build-dist.sh` runs `go build` to produce a single static binary at `dist/curlew`. `dist/` is gitignored — it's a build output, never committed.

## Conventions and constraints

- Optional dependencies (`claude`, `glow`) must degrade gracefully — check with `exec.LookPath` before use.
- AI analysis treats the script as untrusted input: the prompt instructs the model to ignore embedded instructions, the content is fenced with a random sentinel, and `HasInjectionPatterns` blocks analysis unless `--force-analyze` is passed. These guards are backend-agnostic — preserve them when touching the analysis path.
- The AI backend is pluggable via `ResolveCommand` (in `internal/ai`): `CURLEW_AI` selects a preset (`claude`/`ollama`), `CURLEW_MODEL` picks the model, and `CURLEW_AI_CMD` overrides with a raw command. The resolved command receives the prompt on stdin and writes markdown to stdout. A missing/misconfigured backend warns and skips — it never aborts the inspect/execute flow.
- Execution honors the script's shebang via `ValidateShebang` (which rejects multi-arg/unsafe shebangs) and `GetInterpreter`, invoking the interpreter directly rather than piping — this keeps it working on `noexec` /tmp.
- Public AI-config env vars: `CURLEW_AI`, `CURLEW_MODEL`, `CURLEW_AI_CMD`. Test-only env vars (not public API): `CURLEW_SKIP_TTY_CHECK`, `CURLEW_CLAUDE_CMD`.

## Workflow

- Direct pushes to `main` are blocked; all changes go through a PR, one branch per change. CI must be green to merge.
- Architecture decisions are recorded in `docs/decisions/` (ADRs). ADR-008 decided the Go rewrite; check the ADRs before re-litigating a settled design question.
- Releasing: bump `version` in `cmd/curlew/main.go`, commit, tag `vX.Y.Z`, push with `--tags`. The release workflow attaches the binary and a SHA-256 checksum.
