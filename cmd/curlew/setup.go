package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ketsugi/curlew/internal/config"
	"github.com/ketsugi/curlew/internal/setup"
	"github.com/spf13/cobra"
)

func setupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Write the config template and install the shell hook",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSetup()
		},
	}
}

func runSetup() error {
	// --- Config template ---
	if err := writeConfigTemplate(); err != nil {
		// A pre-existing config is fine during setup — note it and continue.
		fmt.Fprintf(os.Stderr, "Config: %s\n", err)
	}

	// --- Shell hook ---
	shell := setup.DetectShell(os.Getenv("SHELL"))
	if shell == "" {
		fmt.Fprintln(os.Stderr, "\nCould not detect a supported shell (zsh or bash).")
		fmt.Fprintln(os.Stderr, "To install the hook manually, add one of these to your shell rc:")
		fmt.Fprintf(os.Stderr, "  %s\n", setup.HookLine("zsh"))
		fmt.Fprintf(os.Stderr, "  %s\n", setup.HookLine("bash"))
		return nil
	}

	rc, err := setup.RCFile(shell)
	if err != nil {
		return err
	}

	installed, err := setup.HookInstalled(rc, shell)
	if err != nil {
		return err
	}
	if installed {
		fmt.Fprintf(os.Stderr, "\nShell hook already installed in %s.\n", rc)
		fmt.Fprintln(os.Stderr, "\nSetup complete.")
		return nil
	}

	fmt.Fprintf(os.Stderr, "\nDetected shell: %s\n", shell)

	// Modifying the user's rc file requires explicit consent. Without an
	// interactive terminal there's no one to consent, so skip rather than
	// default to installing.
	if !interactive() {
		fmt.Fprintln(os.Stderr, "Not an interactive terminal — skipping shell hook installation.")
		fmt.Fprintf(os.Stderr, "To install it, add this to %s:\n  %s\n", rc, setup.HookLine(shell))
		return nil
	}

	fmt.Fprintf(os.Stderr, "Install shell hook? This adds %s to %s\n", setup.HookLine(shell), rc)
	fmt.Fprintf(os.Stderr, "so curl|bash commands are automatically intercepted. [Y/n] ")

	if !promptYes() {
		fmt.Fprintln(os.Stderr, "Skipped hook installation.")
		fmt.Fprintf(os.Stderr, "To install it later, add this to %s:\n  %s\n", rc, setup.HookLine(shell))
		return nil
	}

	if err := setup.InstallHook(rc, shell); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "  Appended hook to: %s\n", rc)
	fmt.Fprintf(os.Stderr, "  Restart your shell or run: source %s\n", rc)
	fmt.Fprintln(os.Stderr, "\nSetup complete.")
	return nil
}

// interactive reports whether stdin is a terminal. The CURLEW_SKIP_TTY_CHECK
// escape hatch (used by tests) forces it true so piped input still drives the
// prompt.
func interactive() bool {
	if os.Getenv("CURLEW_SKIP_TTY_CHECK") == "1" {
		return true
	}
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// promptYes reads a line from stdin and returns true unless the user declines.
// A bare Enter accepts the default (yes); EOF or read error declines, so a
// closed/empty stdin never silently consents to modifying the rc file.
func promptYes() bool {
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil && line == "" {
		return false // EOF with no input — decline
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer != "n" && answer != "no"
}

// writeConfigTemplate writes the commented config template, returning an error
// (not a fatal one) if it already exists.
func writeConfigTemplate() error {
	path := config.FilePath()
	if path == "" {
		return fmt.Errorf("cannot determine config path (no home directory)")
	}

	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("config already exists at %s", path)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	if err := os.WriteFile(path, []byte(config.Template), 0o644); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Wrote config template to: %s\n", path)
	return nil
}
