// Package setup contains the testable logic behind the `curlew setup`
// command: shell detection, rc-file resolution, and idempotent installation
// of the curlew shell hook.
package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// shellSpec describes one shell curlew can install a hook for. Adding a new
// shell is a single entry here (plus the matching hook emitter in
// internal/hook) — DetectShell and RCFile are both driven by this table.
type shellSpec struct {
	// name is the canonical shell identifier passed to `curlew --hook <name>`.
	name string
	// binPrefix matches the basename of $SHELL (e.g. "zsh" matches
	// "zsh", "zsh-5.9", "/opt/homebrew/bin/zsh").
	binPrefix string
	// rcPath resolves the shell's rc file, honoring shell-specific env vars.
	rcPath func() (string, error)
}

var shells = []shellSpec{
	{
		name:      "zsh",
		binPrefix: "zsh",
		rcPath: func() (string, error) {
			// zsh sources $ZDOTDIR/.zshrc when ZDOTDIR is set, else ~/.zshrc.
			if zdotdir := os.Getenv("ZDOTDIR"); zdotdir != "" {
				return filepath.Join(zdotdir, ".zshrc"), nil
			}
			return homeFile(".zshrc")
		},
	},
	{
		name:      "bash",
		binPrefix: "bash",
		rcPath: func() (string, error) {
			return homeFile(".bashrc")
		},
	},
}

// lookupShell finds the spec for a canonical shell name.
func lookupShell(name string) (shellSpec, bool) {
	for _, s := range shells {
		if s.name == name {
			return s, true
		}
	}
	return shellSpec{}, false
}

func homeFile(name string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, name), nil
}

// DetectShell maps a shell path (typically $SHELL) to a supported shell name,
// or "" if the shell is unsupported or the path is empty.
func DetectShell(shellPath string) string {
	if shellPath == "" {
		return ""
	}
	base := filepath.Base(shellPath)
	for _, s := range shells {
		if strings.HasPrefix(base, s.binPrefix) {
			return s.name
		}
	}
	return ""
}

// HookLine returns the line to add to the shell rc file to load the curlew hook.
func HookLine(shell string) string {
	return fmt.Sprintf(`eval "$(curlew --hook %s)"`, shell)
}

// RCFile resolves the rc file path for a given shell.
func RCFile(shell string) (string, error) {
	s, ok := lookupShell(shell)
	if !ok {
		return "", fmt.Errorf("unsupported shell: %s", shell)
	}
	return s.rcPath()
}

// HookInstalled reports whether the curlew hook line is already present in rc.
// A missing file counts as not installed.
func HookInstalled(rc, shell string) (bool, error) {
	data, err := os.ReadFile(rc)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return strings.Contains(string(data), HookLine(shell)), nil
}

// InstallHook appends the curlew hook line to the shell rc file, creating the
// file if it doesn't exist. It is idempotent — if the line is already present,
// it does nothing.
func InstallHook(rc, shell string) error {
	installed, err := HookInstalled(rc, shell)
	if err != nil {
		return err
	}
	if installed {
		return nil
	}

	// Preserve existing content and ensure the hook starts on its own line.
	existing, err := os.ReadFile(rc)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	var b strings.Builder
	if len(existing) > 0 {
		b.Write(existing)
		if existing[len(existing)-1] != '\n' {
			b.WriteByte('\n')
		}
	}
	b.WriteString("\n# Added by curlew setup\n")
	b.WriteString(HookLine(shell))
	b.WriteByte('\n')

	if err := os.MkdirAll(filepath.Dir(rc), 0o755); err != nil {
		return err
	}
	return os.WriteFile(rc, []byte(b.String()), 0o644)
}
