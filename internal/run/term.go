package run

import (
	"fmt"
	"os"

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
// determined.
func termWidth(fallback int) int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 {
		return fallback
	}
	return w
}
