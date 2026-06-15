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

> Coming soon — once [`ketsugi/homebrew-tap`](https://github.com/ketsugi/homebrew-tap) is set up:
> ```bash
> brew install ketsugi/tap/curlew
> ```

### Direct download

```bash
mkdir -p ~/.local/bin
curl -fsSL https://github.com/ketsugi/curlew/releases/latest/download/curlew -o ~/.local/bin/curlew
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
3. Offer AI-powered analysis via [Claude](https://docs.anthropic.com/en/docs/claude-code) (auto-suggested for scripts over 20 lines)
4. Ask for explicit confirmation before executing

You can also point it at a local file:

```bash
curlew ./some-script.sh
```

## Optional dependencies

| Dependency | Purpose | Install |
|------------|---------|---------|
| [claude](https://docs.anthropic.com/en/docs/claude-code) | AI analysis of scripts | `npm install -g @anthropic-ai/claude-code` |
| [glow](https://github.com/charmbracelet/glow) | Rich markdown rendering of analysis output | `brew install glow` |

Both degrade gracefully — curlew works without them, you just won't get the AI analysis or pretty-printed output.

## License

MIT
