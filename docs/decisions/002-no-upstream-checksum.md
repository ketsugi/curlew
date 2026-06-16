# ADR-002: No upstream checksum verification

**Status:** Decided (2026-06-15)
**Issue:** [#6](https://github.com/ketsugi/curlew/issues/6) (closed)

## Context

Considered adding a `--sha256` flag so users could verify a download against a known-good hash.

## Research

Surveyed 60 projects that distribute via `curl | bash`:

| Verification level | Count | % |
|---|---|---|
| Install script itself verified (GPG/checksum) | 2 | ≈3% |
| Script internally verifies binaries it downloads | 13 | ≈22% |
| Checksum exists but curl\|bash path ignores it | 6 | ≈10% |
| TLS-only, no verification | 39 | ≈65% |

Only RVM and mise verify the install script itself. Everyone else relies on HTTPS.

## Decision

Don't implement `--sha256`. The ecosystem doesn't support it.

## Alternative

Invest in #7 (diff/cache mode) instead — curlew becomes its own verification layer since upstreams won't provide one.
