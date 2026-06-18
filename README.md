# curlew

> Inspect before you execute.

A safe wrapper for `curl | bash` that lets you validate, inspect, and optionally AI-analyze scripts before running them.

Named after the [curlew](https://en.wikipedia.org/wiki/Curlew) (a bird), and also a pun on **curl** + re**view**.

## The irony

Yes, one of the install methods involves downloading a script from the internet. We encourage you to vet curlew with itself:

```bash
# After installing, verify what you just put on your system:
curlew "$(which curlew)"
```

We'd respect that.

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
2. Show the line count and offer to open it in `less` for inspection
3. Offer AI-powered analysis via [Claude](https://docs.anthropic.com/en/docs/claude-code) by default, or any backend you configure (auto-suggested for scripts over 20 lines)
4. Ask for explicit confirmation before executing

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

```bash
# Local model via ollama
CURLEW_AI=ollama CURLEW_MODEL=qwen2.5-coder:7b curlew https://example.com/install.sh

# Any other tool: the command receives the prompt on stdin and must
# write the analysis (markdown) to stdout.
CURLEW_AI_CMD="aichat -m openai:gpt-4o" curlew https://example.com/install.sh
```

The backend reads the analysis prompt on stdin and writes markdown to stdout. If the configured backend is missing or misconfigured, curlew warns and skips analysis — it never blocks inspection or execution.

## Configuration file

Instead of exporting env vars in your shell rc, you can persist settings in a TOML config file:

```bash
curlew --init-config   # writes ~/.config/curlew/config.toml
```

```toml
ai = "ollama"
model = "qwen2.5-coder:7b"
threshold = 50   # auto-suggest AI analysis above this many lines
```

Location: `$XDG_CONFIG_HOME/curlew/config.toml` (defaults to `~/.config/curlew/config.toml`).

Precedence: CLI flags > env vars > config file > built-in defaults.

## Shell hook (transparent interception)

Instead of remembering to type `curlew` every time, you can install a shell hook that automatically intercepts `curl ... | bash` commands:

```bash
# zsh (add to ~/.zshrc)
eval "$(curlew --hook zsh)"

# bash (add to ~/.bashrc)
eval "$(curlew --hook bash)"
```

With the hook active, any `curl | bash`, `curl | sh`, `wget | bash`, or `wget | sh` command you type will be intercepted and routed through curlew for inspection before execution.

Note: the hook intercepts pipe-to-shell only. Process substitution (`bash <(curl ...)`), eval forms (`eval "$(curl ...)"`), and two-step downloads (`curl -o file && bash file`) are not covered.

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
