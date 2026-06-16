# ADR-003: Diff/cache mode is worth building

**Status:** Validated (2026-06-15)
**Issue:** [#7](https://github.com/ketsugi/curlew/issues/7) (open)

## Context

Should curlew cache previously-approved scripts and diff against them on re-run?

## Research

Surveyed 47 projects for their update story:

| Update method | Count | % |
|---|---|---|
| Built-in self-update | 28 | ≈60% |
| Re-run same curl\|bash command | 17 | ≈36% |
| Package manager only | 2 | ≈4% |

≈36% of the ecosystem tells users to re-run the install script for updates (nvm, Starship, k3s, Helm, Volta, Pulumi, etc.).

## Decision

This feature has a validated use case. Users re-running URLs weeks later have no way to know if the script changed. Curlew's diff mode catches that.

## Design

- First run: user inspects and approves → curlew caches content hash + optionally full script
- Re-run same URL: compare against cache
  - Unchanged: note it, skip to execute prompt
  - Changed: show diff, proceed with normal inspection flow
- Cache location: `~/.cache/curlew/` with URL as key
