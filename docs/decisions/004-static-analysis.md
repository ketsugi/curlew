# ADR-004: Static analysis is feasible with grep

**Status:** Validated (2026-06-15)
**Issue:** [#8](https://github.com/ketsugi/curlew/issues/8) (open)

## Context

Can we offer a "dry-run" mode that shows what a script will do without executing it?

## Research

No built-in macOS/Linux tool performs semantic static analysis of shell scripts. However, grep-based pattern matching works well for installer scripts because they tend to use commands literally (not via variables).

## Prototype results

Tested a ≈50-line grep-based analyzer on nvm's install script (500+ lines). Correctly identified:
- Network calls (curl, wget, git fetch)
- URLs (raw.githubusercontent.com)
- File writes (mkdir, cp, >> to profile)
- Privilege escalation (chmod)
- Persistence (modifies .bashrc/.zshrc)
- Obfuscation (eval wget)

## Decision

Implement Tier 1 (grep-based, zero deps, ≈50 lines). It surfaces ≈80% of script behavior for the "should I run this?" decision.

## Limitations

- Can't follow variables or resolve conditionals
- Comments and heredocs cause some false positives (fixable with better filtering)
- Not a replacement for manual inspection, but tells you where to look

## Future tiers

- Tier 2: AST parsing via `shfmt` (optional dependency)
- Tier 3: Compiled companion binary for full analysis
