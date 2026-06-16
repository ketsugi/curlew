# ADR-001: No dedicated landing page

**Status:** Decided (2026-06-15)
**Issue:** [#3](https://github.com/ketsugi/curlew/issues/3) (closed)

## Context

Considered whether a GitHub Pages site would help discoverability.

## Decision

The README is the landing page. No separate site needed.

## Rationale

- Project is a ≈200-line bash script with a single purpose
- The README covers install + usage in under a screenful
- GitHub renders it well and it's where developers land
- A separate site is a maintenance burden that says the same thing with prettier CSS
- Can revisit if discoverability becomes a real problem (it isn't yet)
