# ADR-007: Stay bash (for now)

**Status:** Decided (2026-06-15)

## Context

The dist build script complexity (inlining hooks via sed heredoc patching) raised the question: should curlew be rewritten in a compiled language?

## Decision

Stay bash for now. The current feature set (download, inspect, confirm, AI analysis) is within bash's comfort zone. Draw a line at the next complexity threshold.

## Rewrite trigger

Rewrite when implementing any two of:
- #4: Configurable AI harness with template expansion
- #7: Diff/cache mode with persistent store
- #8: Static analysis with structured output

Any two of these together would make bash actively hostile.

## Language comparison

| | Go | Rust | Node |
|---|---|---|---|
| Single binary | ✅ Easy cross-compile | ✅ Smallest binaries | ❌ Needs runtime |
| Homebrew | ✅ Bottles | ✅ Bottles | ⚠️ Heavy |
| Unicode/homograph (#9) | ✅ Good stdlib | ✅ `unicode_skeleton` crate | ✅ ICU |
| Dev speed | ✅ Fast | ⚠️ Slower | ✅ Fastest |
| CLI ecosystem | ✅ cobra/viper | ✅ clap | ❌ node_modules |
| Startup time | ✅ Instant | ✅ Instant | ❌ ~100ms |
| Audience fit | ✅ DevOps | ✅ Performance | ❌ Wrong crowd |

## Recommendation if rewriting

- **Go** for fast iteration, easy cross-compilation, DevOps CLI conventions
- **Rust** for smallest binary, best performance, `unicode_skeleton` for homograph detection (#9)
- **Not Node** — the audience installs things via `curl | bash`, they shouldn't need Node to protect against it

## What bash is still good for

- The shell hooks (zsh preexec / bash DEBUG trap) will always be shell code regardless of the main binary's language — you `eval` shell code into the user's shell
- Rapid prototyping of new detection patterns
- Zero-dependency installs via the single-file dist artifact
