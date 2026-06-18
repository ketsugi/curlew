# Development

## Prerequisites

- Go 1.21+
- [bats-core](https://github.com/bats-core/bats-core) (for integration tests)

```bash
brew install go bats-core   # macOS / Linuxbrew
apt install golang bats     # Debian/Ubuntu
```

## Project structure

```
curlew/
├── cmd/curlew/main.go            # entrypoint (cobra CLI)
├── internal/
│   ├── hook/                     # shell hook string constants + tests
│   ├── validate/                 # pure validation functions + tests
│   ├── ai/                       # AI backend resolution + tests
│   └── run/                      # interactive flow + terminal helpers
├── scripts/
│   ├── build-dist.sh             # builds dist/curlew via go build
│   ├── test-go.sh                # build + all tests (unit + integration)
│   └── dev-shell.sh              # isolated shell with in-repo hook loaded
├── test/
│   ├── curlew-integration.bats   # end-to-end flow tests
│   └── hook.bats                 # shell-hook emitter tests
├── .github/workflows/
│   ├── ci.yml                    # tests on PR/push to main
│   └── release.yml               # tag-triggered GitHub Release
├── docs/decisions/               # architecture decision records (ADRs)
├── go.mod
├── go.sum
├── README.md
├── DEVELOPMENT.md                # this file
└── LICENSE
```

## Running tests

Full suite (build + Go unit tests + bats integration):

```bash
scripts/test-go.sh
```

Go unit tests only:

```bash
go test ./...
```

Integration tests only (requires the binary to be built first):

```bash
go build -o bin/curlew-go ./cmd/curlew/
bats test/
```

## Building the dist artifact

```bash
scripts/build-dist.sh             # writes dist/curlew
scripts/build-dist.sh /tmp/curlew # or a custom output path
```

`dist/` is gitignored — it's a build output, never committed.

## Running curlew locally

```bash
go build -o bin/curlew-go ./cmd/curlew/
bin/curlew-go https://example.com/install.sh
bin/curlew-go ./some-local-script.sh
```

## Testing the shell hook locally

The hook emitters (`curlew --hook zsh|bash`) generate code that runs in your live shell. Two layers:

**Automated.** `bats test/hook.bats` evals the emitted hook in its target shell and fires the interception function directly — covering parse-cleanliness, `curl|bash` routing, sudo skip, and `CURLEW_BYPASS`.

**Manual.** To type real `curl ... | bash` commands against your working copy:

```bash
scripts/dev-shell.sh        # zsh (default)
scripts/dev-shell.sh bash
```

This launches an interactive shell with the in-repo binary on PATH and the hook loaded, isolated from your real shell config.

## Adding new testable logic

1. Write the function in the appropriate `internal/` package
2. Add Go tests in the same package (`_test.go` file)
3. If it affects observable CLI behavior, add/update a bats test
4. Run `scripts/test-go.sh` to verify

## Environment variables (test-only internals)

These exist to support automated testing. They are not part of the public interface and may change without notice.

| Variable | Purpose |
|----------|---------|
| `CURLEW_SKIP_TTY_CHECK` | Set to `1` to bypass the interactive terminal check (allows piped stdin in tests) |
| `CURLEW_CLAUDE_CMD` | Override the claude binary path (tests point this at a mock stub) |

## Contributing

All changes go through pull requests — direct pushes to `main` are blocked.

1. Create a branch: `git checkout -b my-feature`
2. Make changes, run `scripts/test-go.sh` locally
3. Push and open a PR: `git push -u origin my-feature`
4. CI runs automatically; merge once green

## Releasing

1. Bump `version` in `cmd/curlew/main.go`
2. Commit: `git commit -am "chore: bump version to X.Y.Z"`
3. Tag: `git tag vX.Y.Z`
4. Push: `git push origin main --tags`

The GitHub Actions workflow builds the binary for all platforms and attaches them with SHA-256 checksums.
