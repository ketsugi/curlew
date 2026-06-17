# ADR-008: Rewrite curlew in Go

**Status:** Decided (2026-06-17)
**Issue:** [#28](https://github.com/ketsugi/curlew/issues/28) (open)
**Supersedes:** [ADR-007](007-bash-vs-compiled.md)

## Context

ADR-007 kept curlew in bash and set a rewrite trigger: implementing any two of #4 (configurable AI harness), #7 (diff/cache mode), or #8 (static analysis). Since then:

- **#4 shipped** ([#22](https://github.com/ketsugi/curlew/issues/4)) — but the version that landed deliberately dodged the bash-hostile part. There is no template-expansion engine; backends use a stdin/stdout contract plus presets (`CURLEW_AI` / `CURLEW_MODEL` / `CURLEW_AI_CMD`). So #4 as built stayed inside bash's comfort zone.
- **#7 folded into a ledger epic** ([#23](https://github.com/ketsugi/curlew/issues/23)) — `curlew list` / `update` / `uninstall` on top of a persistent install ledger, with the binary identity captured by before/after filesystem diffing rather than script parsing (see #7 investigation).

The trigger is therefore met more in spirit than by a clean ticket count. The real driver is the shape of the upcoming work, not the tally:

- The ledger (#23/#24) is persistent structured state plus before/after filesystem diffing — the class of thing bash makes tedious and hard to test.
- #8 (static analysis with structured output) and #9 (unicode/homograph detection) need real libraries bash does not have.

This is the complexity threshold ADR-007 was drawing a line at.

## Decision

Rewrite curlew in **Go**, before further feature work on #23/#7/#8/#9. The rewrite is the prerequisite; building the ledger in bash and then porting it would be wasted effort.

Not Rust, not Node:

| | Why |
|---|---|
| **Go (chosen)** | A subprocess-orchestrating CLI with subcommands and filesystem work — not a perf problem. cobra/viper give the `list`/`update`/`uninstall` surface; cross-compile and Homebrew bottles are trivial; the audience is DevOps. |
| Rust (rejected) | Wins only if #9's homograph detection becomes a centerpiece (`unicode_skeleton`) and smallest-binary/perf outweigh dev speed. They don't here. |
| Node (rejected) | The audience installs things via `curl \| bash`; they should not need a Node runtime to protect against it. |

## Approach — incremental port, not big-bang

1. Stand up the Go skeleton (cobra) with the existing subcommand/flag surface.
2. Port the pure core from `lib/curlew-lib.sh` — `validate_mimetype`, `has_null_bytes`, `has_injection_patterns`, `validate_shebang`, `get_interpreter`, `resolve_ai_command` — to tested Go functions. They are already side-effect-free and map almost mechanically.
3. Reach parity on the interactive flow (download → inspect → confirm → execute), using the existing `bats` suite as the behavioral spec.
4. Build the new work natively in Go: ledger + footprint diff (#24), diff/re-vet (#7), `update` (#26), `uninstall` (#27), then #8/#9.

## What stays shell

The zsh `preexec` / bash `DEBUG`-trap hooks remain shell code emitted by `curlew --hook` — you `eval` shell into the user's live shell regardless of the binary's language. The Go binary keeps emitting them. curlew is therefore always a Go-binary-plus-shell-snippets project, never pure Go.

## Invariants the rewrite must preserve

- The zero-dependency single-file install story (a Go binary is *more* self-contained — no lib-inlining `sed` dance) and the Homebrew tap release pipeline.
- The security guards: mimetype / null-byte / shebang validation, the injection-pattern gate on AI analysis, sentinel fencing, interpreter-direct execution for `noexec` /tmp.
- Test coverage — port the bats behaviors to Go tests as the parity gate before adding features.

## Consequences

- Short term, feature delivery pauses while the port reaches parity; the working download→inspect→confirm→execute path gains nothing from Go and carries regression risk during the move.
- Long term, the ledger, static analysis, and unicode work become tractable and testable, and the install/release story simplifies (no dist inlining).
