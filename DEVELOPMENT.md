# Development

## Prerequisites

- bash 4.0+
- [bats-core](https://github.com/bats-core/bats-core) (for tests)

```bash
brew install bats-core   # macOS / Linuxbrew
apt install bats         # Debian/Ubuntu
```

## Project structure

```
curlew/
├── bin/curlew              # main entrypoint
├── lib/curlew-lib.sh       # testable functions (sourced by bin/curlew)
├── test/
│   ├── test_helper.bash    # bats test setup
│   └── curlew-lib.bats     # unit tests
├── .github/workflows/
│   └── release.yml         # tag-triggered GitHub Release
├── README.md
├── DEVELOPMENT.md          # this file
└── LICENSE
```

## Running tests

```bash
bats test/
```

Or a specific file:

```bash
bats test/curlew-lib.bats
```

## Running curlew locally

From the repo root:

```bash
bin/curlew https://example.com/install.sh
bin/curlew ./some-local-script.sh
```

## How the lib sourcing works

`bin/curlew` sources `lib/curlew-lib.sh` via:

```bash
CURLEW_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
source "${CURLEW_LIB:-$CURLEW_ROOT/lib/curlew-lib.sh}"
```

The `CURLEW_LIB` env var lets tests (and alternate install layouts) point at the lib directly.

## Adding new testable logic

1. Write the function in `lib/curlew-lib.sh`
2. Use it in `bin/curlew`
3. Add tests in `test/curlew-lib.bats`
4. Run `bats test/` to verify

## Environment variables (test-only internals)

These exist to support automated testing. They are not part of the public interface and may change without notice.

| Variable | Purpose |
|----------|---------|
| `CURLEW_LIB` | Override path to `curlew-lib.sh` (used by bats test helper) |
| `CURLEW_SKIP_TTY_CHECK` | Set to `1` to bypass the interactive terminal check (allows piped stdin in tests) |
| `CURLEW_CLAUDE_CMD` | Override the claude binary path (tests point this at a mock stub) |
| `CURLEW_MODEL` | Override the Claude model name (default: `sonnet`) |

## Releasing

1. Bump `VERSION` in `bin/curlew`
2. Commit: `git commit -am "chore: bump version to X.Y.Z"`
3. Tag: `git tag vX.Y.Z`
4. Push: `git push origin main --tags`

The GitHub Actions workflow creates a release with `bin/curlew` and a SHA-256 checksum attached.
