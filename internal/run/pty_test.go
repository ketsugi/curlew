package run

import (
	"os"

	"github.com/creack/pty"
)

// openPTY creates a pseudo-terminal pair and returns the master side.
func openPTY() (*os.File, error) {
	master, _, err := pty.Open()
	if err != nil {
		return nil, err
	}
	return master, nil
}

// setPTYSize sets the terminal size on a PTY fd.
func setPTYSize(f *os.File, cols, rows int) error {
	return pty.Setsize(f, &pty.Winsize{
		Cols: uint16(cols),
		Rows: uint16(rows),
	})
}
