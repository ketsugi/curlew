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
├── bin/curlew                    # main entrypoint
├── lib/curlew-lib.sh             # testable functions (sourced by bin/curlew)
├── scripts/build-dist.sh         # inlines lib into bin for a single-file release
├── test/
│   ├── test_helper.bash          # bats test setup
│   ├── curlew-lib.bats           # unit tests for lib functions
│   ├── curlew-integration.bats   # end-to-end flow tests
│   └── hook.bats                 # shell-hook emitter tests
├── .github/workflows/
│   ├── ci.yml                    # tests + dist smoke test on PR/push to main
│   └── release.yml               # tag-triggered GitHub Release
├── docs/decisions/               # architecture decision records (ADRs)
├── README.md
├── DEVELOPMENT.md                # this file
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

CI (`.github/workflows/ci.yml`) runs `bats test/` on every PR and push to `main`, then builds the single-file dist artifact and smoke-tests it.

## Building the dist artifact

The release is a single self-contained file with `lib/curlew-lib.sh` inlined into `bin/curlew`:

```bash
scripts/build-dist.sh             # writes dist/curlew
scripts/build-dist.sh /tmp/curlew # or a custom output path
```

`dist/` is gitignored — it's a build output, never committed. The lib stays separate in the repo so it remains unit-testable.

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

## Contributing

All changes go through pull requests — direct pushes to `main` are blocked.

1. Create a branch: `git checkout -b my-feature`
2. Make changes, run `bats test/` locally
3. Push and open a PR: `git push -u origin my-feature`
4. CI runs automatically; merge once green

## Releasing

1. Bump `VERSION` in `bin/curlew`
2. Commit: `git commit -am "chore: bump version to X.Y.Z"`
3. Tag: `git tag vX.Y.Z`
4. Push: `git push origin main --tags`

The GitHub Actions workflow creates a release with `bin/curlew` and a SHA-256 checksum attached.
