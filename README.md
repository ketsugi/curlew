# curlew

[![codecov](https://codecov.io/gh/ketsugi/curlew/graph/badge.svg?token=76DKXGQBDU)](https://codecov.io/gh/ketsugi/curlew)

> Inspect before you execute.

A safe wrapper for `curl | bash` that lets you validate, inspect, and optionally AI-analyze scripts before running them.

Named after the [curlew](https://en.wikipedia.org/wiki/Curlew) (a bird), and also a pun on **curl** + re**view**.

## Install

### Homebrew (macOS & Linux)

```bash
brew install ketsugi/tap/curlew
```

### Direct download

```bash
mkdir -p ~/.local/bin

# macOS (Apple Silicon)
curl -fsSL https://github.com/ketsugi/curlew/releases/latest/download/curlew-darwin-arm64 -o ~/.local/bin/curlew

# macOS (Intel)
curl -fsSL https://github.com/ketsugi/curlew/releases/latest/download/curlew-darwin-amd64 -o ~/.local/bin/curlew

# Linux (x86_64)
curl -fsSL https://github.com/ketsugi/curlew/releases/latest/download/curlew-linux-amd64 -o ~/.local/bin/curlew

# Linux (ARM64)
curl -fsSL https://github.com/ketsugi/curlew/releases/latest/download/curlew-linux-arm64 -o ~/.local/bin/curlew

chmod +x ~/.local/bin/curlew
```

Make sure `~/.local/bin` is in your `PATH`.

## Usage

```bash
# Instead of: curl -fsSL https://example.com/install.sh | bash
# Do:
curlew https://example.com/install.sh
```

curlew will:

1. Download the script and verify it's actually a text-based script (not a binary)
2. Run a fast structural analysis (network calls, file writes, package installs, privilege escalation, persistence, dangerous ops) — deterministic, no dependencies, always on
3. Show the line count and offer to open it in `less` for inspection
4. Offer AI-powered analysis via [Claude](https://docs.anthropic.com/en/docs/claude-code) by default, or any backend you configure (auto-suggested for scripts over 20 lines)
5. Ask for explicit confirmation before executing

The structural analysis is the deterministic floor — it tells you literally what's in the script regardless of whether you have an AI backend configured. The AI pass adds a semantic judgment on top.

You can also point it at a local file:

```bash
curlew ./some-script.sh
```

## Choosing the AI backend

The AI analysis defaults to the `claude` CLI, but you can point curlew at any tool via environment variables:

| Variable | Purpose | Default |
|----------|---------|---------|
| `CURLEW_AI` | Backend preset: `claude` or `ollama` | `claude` |
| `CURLEW_MODEL` | Model name passed to the preset | `sonnet` (claude); required for `ollama` |
| `CURLEW_AI_CMD` | Raw command override; wins over any preset | — |
| `CURLEW_THRESHOLD` | Auto-suggest AI analysis above this many lines | `20` |

```bash
# Local model via ollama
CURLEW_AI=ollama CURLEW_MODEL=qwen2.5-coder:7b curlew https://example.com/install.sh

# Any other tool: the command receives the prompt on stdin and must
# write the analysis (markdown) to stdout.
CURLEW_AI_CMD="aichat -m openai:gpt-4o" curlew https://example.com/install.sh
```

The backend reads the analysis prompt on stdin and writes markdown to stdout. If the configured backend is missing or misconfigured, curlew warns and skips analysis — it never blocks inspection or execution.

## Setup

```bash
curlew setup
```

`curlew setup` writes a config template and offers to install the shell hook into your rc file (it detects zsh or bash from `$SHELL`). It's idempotent — safe to run again; it won't duplicate the hook or overwrite an existing config.

## Configuration file

Instead of exporting env vars in your shell rc, you can persist settings in a TOML config file. `curlew setup` writes a commented template to `~/.config/curlew/config.toml`:

```toml
ai = "ollama"
model = "qwen2.5-coder:7b"
threshold = 50   # auto-suggest AI analysis above this many lines
```

Location: `$XDG_CONFIG_HOME/curlew/config.toml` (defaults to `~/.config/curlew/config.toml`).

Precedence: CLI flags > env vars > config file > built-in defaults.

## Shell hook (transparent interception)

Instead of remembering to type `curlew` every time, you can install a shell hook that automatically intercepts `curl ... | bash` commands. The easiest way is `curlew setup`, which detects your shell and installs it for you. To do it manually:

```bash
# zsh (add to ~/.zshrc)
eval "$(curlew --hook zsh)"

# bash (add to ~/.bashrc)
eval "$(curlew --hook bash)"
```

With the hook active, two common forms are recognized:

- **Pipe-to-shell** — `curl ... | bash`, `curl ... | sh`, `wget ... | bash`, etc. Intercepted and rerouted through curlew before anything runs (both zsh and bash).
- **Command substitution** — `bash -c "$(curl ...)"` (the Homebrew installer's form). In **zsh** this is intercepted before the download runs. In **bash** it can't be blocked — the shell performs the substitution (running the download) before the hook can act — so curlew prints a warning with a pastable `curlew <url>` command to vet the script before its contents are trusted.

Not covered: process substitution (`bash <(curl ...)`), `eval "$(curl ...)"`, and two-step download-then-run (`curl -o file && bash file`).

To bypass the hook for a single command:

```bash
CURLEW_BYPASS=1 curl -fsSL https://example.com/install.sh | bash
```

## Optional dependencies

| Dependency | Purpose | Install |
|------------|---------|---------|
| [claude](https://docs.anthropic.com/en/docs/claude-code) | AI analysis (default backend) | `npm install -g @anthropic-ai/claude-code` |
| [ollama](https://ollama.com) | AI analysis (local models) | `brew install ollama` |
| [glow](https://github.com/charmbracelet/glow) | Rich markdown rendering of analysis output | `brew install glow` |

All degrade gracefully — curlew works without them, you just won't get the AI analysis or pretty-printed output.

## Output streams

curlew writes its own diagnostics (progress messages, warnings, prompts) to **stderr**. The AI analysis markdown goes to **stdout**. This means you can capture just the analysis:

```bash
curlew https://example.com/install.sh 2>/dev/null > analysis.md
```

## License

MIT
