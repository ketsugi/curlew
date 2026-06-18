package run

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/term"
)

// confirm prints a prompt and reads a single keypress (no Enter needed).
// Returns true for y/Y, false for n/N. A bare Enter uses the default.
func confirm(prompt string, defaultYes bool) (bool, error) {
	fmt.Fprint(os.Stderr, prompt)

	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		// Fallback: can't make raw, read a line
		return confirmFallback(defaultYes)
	}
	defer term.Restore(fd, oldState)

	buf := make([]byte, 1)
	n, err := os.Stdin.Read(buf)

	// Restore before printing newline
	term.Restore(fd, oldState)

	if err != nil || n == 0 {
		fmt.Fprintln(os.Stderr)
		return defaultYes, nil
	}

	key := buf[0]
	switch key {
	case '\r', '\n':
		if defaultYes {
			fmt.Fprintln(os.Stderr, "y")
		} else {
			fmt.Fprintln(os.Stderr, "n")
		}
		return defaultYes, nil
	case 'y', 'Y':
		fmt.Fprintln(os.Stderr, "y")
		return true, nil
	case 'n', 'N':
		fmt.Fprintln(os.Stderr, "n")
		return false, nil
	case 3: // Ctrl-C
		fmt.Fprintln(os.Stderr)
		os.Exit(130)
		return false, nil
	default:
		fmt.Fprintln(os.Stderr, string(key))
		return defaultYes, nil
	}
}

func confirmFallback(defaultYes bool) (bool, error) {
	buf := make([]byte, 1)
	n, _ := os.Stdin.Read(buf)
	if n == 0 {
		return defaultYes, nil
	}
	switch buf[0] {
	case 'y', 'Y':
		fmt.Fprintln(os.Stderr, "y")
		return true, nil
	case 'n', 'N':
		fmt.Fprintln(os.Stderr, "n")
		return false, nil
	case '\r', '\n':
		if defaultYes {
			fmt.Fprintln(os.Stderr, "y")
		} else {
			fmt.Fprintln(os.Stderr, "n")
		}
		return defaultYes, nil
	default:
		fmt.Fprintln(os.Stderr, string(buf[0]))
		return defaultYes, nil
	}
}

// termWidth returns the current terminal width, or fallback if it can't be
// determined. Tries multiple sources: stdout, stderr, /dev/tty, and the
// COLUMNS env var (works inside pipelines and under `script`).
func termWidth(fallback int) int {
	// Try stdout
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
		return w
	}
	// Try stderr (often still attached to the terminal when stdout is piped)
	if w, _, err := term.GetSize(int(os.Stderr.Fd())); err == nil && w > 0 {
		return w
	}
	// Try /dev/tty
	if tty, err := os.Open("/dev/tty"); err == nil {
		defer tty.Close()
		if w, _, err := term.GetSize(int(tty.Fd())); err == nil && w > 0 {
			return w
		}
	}
	// Try stty size via /dev/tty (matches the bash version's approach;
	// reads the line discipline which reflects stty cols changes)
	if tty, err := os.Open("/dev/tty"); err == nil {
		cmd := exec.Command("stty", "size")
		cmd.Stdin = tty
		if out, err := cmd.Output(); err == nil {
			fields := strings.Fields(strings.TrimSpace(string(out)))
			if len(fields) >= 2 {
				if w := parseInt(fields[1]); w > 0 {
					tty.Close()
					return w
				}
			}
		}
		tty.Close()
	}
	// Try COLUMNS env var
	if cols := os.Getenv("COLUMNS"); cols != "" {
		if w := parseInt(cols); w > 0 {
			return w
		}
	}
	return fallback
}

func parseInt(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}
