# ADR-005: Configuration system design

**Status:** Superseded (2026-06-18) — implemented as TOML per #33; see `internal/config/`
**Issues:** [#4](https://github.com/ketsugi/curlew/issues/4), [#5](https://github.com/ketsugi/curlew/issues/5), [#33](https://github.com/ketsugi/curlew/issues/33)

## Context

Both the configurable AI backend (#4) and configurable threshold (#5) need a unified config system.

## Decision

Three-layer precedence: CLI flags > env vars > config file > defaults.

## Config file

Location: `$XDG_CONFIG_HOME/curlew/config` (defaults to `~/.config/curlew/config`)

Format: plain bash variable assignments, sourced directly:

```bash
threshold=50
ai_cmd=kiro
ai_args="chat -p"
model=sonnet
```

## Environment variables

Namespaced with `CURLEW_` prefix:

```bash
CURLEW_THRESHOLD=50
CURLEW_AI_CMD=claude
CURLEW_AI_ARGS="--print"
CURLEW_MODEL=sonnet
```

## CLI flags

```bash
curlew --threshold 50 --ai-cmd kiro --model opus https://example.com
```

## Migration

- `CURLEW_CLAUDE_CMD` → `CURLEW_AI_CMD` (honour old name for one release cycle)
- Config file is optional — curlew works with defaults
- `curlew --init-config` generates a commented template

## AI harness design

Generic `ai_cmd` + `ai_args` approach — users plug in any tool that accepts a prompt. No hardcoded presets, just documented examples in the README.
