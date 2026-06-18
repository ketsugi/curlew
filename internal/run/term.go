package run

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/term"
)

// ErrInterrupted is returned by confirm when the user hits Ctrl-C at a prompt.
// Raw mode suppresses SIGINT (it delivers byte 3 instead), so this is the
// in-band signal that lets Execute return normally and run its deferred
// temp-file cleanup, rather than os.Exit skipping it.
var ErrInterrupted = errors.New("interrupted")

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

	key, ok := readKeypress(os.Stdin)

	// Restore before printing newline
	term.Restore(fd, oldState)

	if !ok {
		fmt.Fprintln(os.Stderr)
		return defaultYes, nil
	}

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
	case 3: // Ctrl-C (raw mode delivers ETX, not SIGINT)
		fmt.Fprintln(os.Stderr)
		return false, ErrInterrupted
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
	case 3: // Ctrl-C
		fmt.Fprintln(os.Stderr)
		return false, ErrInterrupted
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

// readKeypress reads one logical keypress from r and returns its first byte.
// Reading into a small buffer (rather than a single byte) drains the trailing
// bytes of a multi-byte sequence — e.g. an arrow key arrives as ESC '[' 'A' in
// raw mode — so they don't leak into the next prompt or back to the shell. ok
// is false when nothing was read.
func readKeypress(r io.Reader) (key byte, ok bool) {
	buf := make([]byte, 8)
	n, _ := r.Read(buf)
	if n == 0 {
		return 0, false
	}
	return buf[0], true
}

// termWidth returns the current terminal width, or fallback if it can't be
// determined. Tries multiple sources: stdout, stderr, /dev/tty, stty, and the
// COLUMNS env var (works inside pipelines and under `script`).
func termWidth(fallback int) int {
	return termWidthFromFds([]uintptr{os.Stdout.Fd(), os.Stderr.Fd()}, fallback)
}

// termWidthFromFds probes the given fds for terminal width, then falls back to
// /dev/tty, stty, COLUMNS, and finally the provided fallback value.
func termWidthFromFds(fds []uintptr, fallback int) int {
	for _, fd := range fds {
		if w, _, err := term.GetSize(int(fd)); err == nil && w > 0 {
			return w
		}
	}
	// Try /dev/tty
	if tty, err := os.Open("/dev/tty"); err == nil {
		w, _, err := term.GetSize(int(tty.Fd()))
		tty.Close()
		if err == nil && w > 0 {
			return w
		}
	}
	// Try stty size via /dev/tty (matches the bash version's approach;
	// reads the line discipline which reflects stty cols changes)
	if tty, err := os.Open("/dev/tty"); err == nil {
		cmd := exec.Command("stty", "size")
		cmd.Stdin = tty
		out, err := cmd.Output()
		tty.Close()
		if err == nil {
			fields := strings.Fields(strings.TrimSpace(string(out)))
			if len(fields) >= 2 {
				if w := parseInt(fields[1]); w > 0 {
					return w
				}
			}
		}
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
