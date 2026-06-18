package run

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestReadKeypress_SingleByte(t *testing.T) {
	r := bytes.NewReader([]byte{'y'})
	key, ok := readKeypress(r)
	if !ok || key != 'y' {
		t.Fatalf("expected ('y', true), got (%#x, %v)", key, ok)
	}
}

func TestReadKeypress_Empty(t *testing.T) {
	r := bytes.NewReader(nil)
	if _, ok := readKeypress(r); ok {
		t.Error("expected ok=false on empty input")
	}
}

func TestReadKeypress_DrainsEscapeSequence(t *testing.T) {
	// An arrow key is the 3-byte sequence ESC '[' 'A'. readKeypress must return
	// the first byte AND consume the trailing bytes, so they don't leak into the
	// next read (the bug in #40, where only 1 byte was consumed).
	r := bytes.NewReader([]byte{0x1b, '[', 'A'})
	key, ok := readKeypress(r)
	if !ok || key != 0x1b {
		t.Fatalf("expected (0x1b, true), got (%#x, %v)", key, ok)
	}
	rest := make([]byte, 4)
	if n, _ := r.Read(rest); n != 0 {
		t.Errorf("escape sequence leaked %d trailing bytes: %v", n, rest[:n])
	}
}

// TestConfirmFallback_CtrlC verifies that byte 3 (Ctrl-C) in the pipe fallback
// path returns ErrInterrupted rather than being treated as a default answer.
func TestConfirmFallback_CtrlC(t *testing.T) {
	origStdin := os.Stdin
	origStderr := os.Stderr
	defer func() {
		os.Stdin = origStdin
		os.Stderr = origStderr
	}()

	// Pipe Ctrl-C (byte 3) as stdin
	r, w, _ := os.Pipe()
	w.Write([]byte{3})
	w.Close()
	os.Stdin = r

	// Suppress stderr output from the prompt
	_, sw, _ := os.Pipe()
	os.Stderr = sw

	_, err := confirmFallback(true)
	sw.Close()

	if err != ErrInterrupted {
		t.Errorf("expected ErrInterrupted, got: %v", err)
	}
}

// TestConfirmFallback_DefaultYes verifies Enter uses the default.
func TestConfirmFallback_DefaultYes(t *testing.T) {
	origStdin := os.Stdin
	origStderr := os.Stderr
	defer func() {
		os.Stdin = origStdin
		os.Stderr = origStderr
	}()

	r, w, _ := os.Pipe()
	w.Write([]byte{'\n'})
	w.Close()
	os.Stdin = r

	_, sw, _ := os.Pipe()
	os.Stderr = sw

	yes, err := confirmFallback(true)
	sw.Close()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !yes {
		t.Error("expected true (default yes), got false")
	}
}

// TestConfirmFallback_ExplicitNo verifies 'n' overrides a yes default.
func TestConfirmFallback_ExplicitNo(t *testing.T) {
	origStdin := os.Stdin
	origStderr := os.Stderr
	defer func() {
		os.Stdin = origStdin
		os.Stderr = origStderr
	}()

	r, w, _ := os.Pipe()
	w.Write([]byte{'n'})
	w.Close()
	os.Stdin = r

	_, sw, _ := os.Pipe()
	os.Stderr = sw

	yes, err := confirmFallback(true)
	sw.Close()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if yes {
		t.Error("expected false for explicit 'n', got true")
	}
}

func TestParseInt_Valid(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"0", 0},
		{"42", 42},
		{"160", 160},
	}
	for _, tc := range cases {
		if got := parseInt(tc.in); got != tc.want {
			t.Errorf("parseInt(%q) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestParseInt_Invalid(t *testing.T) {
	for _, in := range []string{"", "abc", "12x", "-5"} {
		if got := parseInt(in); got != 0 {
			t.Errorf("parseInt(%q) = %d, want 0", in, got)
		}
	}
}

func TestTermWidth_FallsBackToColumns(t *testing.T) {
	t.Setenv("COLUMNS", "200")
	// Pass no valid fds so all fd-based checks fail and we hit COLUMNS.
	w := termWidthFromFds(nil, 80)
	if w != 200 {
		t.Errorf("expected 200 from COLUMNS, got %d", w)
	}
}

func TestTermWidth_InvalidColumns(t *testing.T) {
	t.Setenv("COLUMNS", "notanumber")
	w := termWidthFromFds(nil, 80)
	if w != 80 {
		t.Errorf("expected fallback 80, got %d", w)
	}
}

func TestTermWidth_FromPTY(t *testing.T) {
	pty, err := openPTY()
	if err != nil {
		t.Skipf("cannot open PTY: %v", err)
	}
	defer pty.Close()

	// Set the PTY to a known width
	if err := setPTYSize(pty, 142, 40); err != nil {
		t.Skipf("cannot set PTY size: %v", err)
	}

	w := termWidthFromFds([]uintptr{pty.Fd()}, 80)
	if w != 142 {
		t.Errorf("expected 142 from PTY, got %d", w)
	}
}

func TestTermWidth_FallbackWhenFdNotTerminal(t *testing.T) {
	// A pipe fd is not a terminal — termWidthFromFds should skip it
	r, w, _ := os.Pipe()
	defer r.Close()
	defer w.Close()

	t.Setenv("COLUMNS", "")
	got := termWidthFromFds([]uintptr{r.Fd()}, 99)
	// Should fall through to fallback (no COLUMNS set, pipe isn't a terminal)
	if got == 0 {
		t.Error("should not return 0")
	}
}

// TestCountLines verifies line counting edge cases.
func TestCountLines(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    int
	}{
		{"empty", "", 0},
		{"one line no newline", "hello", 1},
		{"one line with newline", "hello\n", 1},
		{"two lines", "a\nb\n", 2},
		{"trailing no newline", "a\nb", 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := filepath.Join(t.TempDir(), "f")
			os.WriteFile(p, []byte(tc.content), 0o644)
			got, err := countLines(p)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("countLines(%q) = %d, want %d", tc.content, got, tc.want)
			}
		})
	}
}

func TestFirstLine(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"#!/bin/bash\necho hi\n", "#!/bin/bash"},
		{"single line no newline", "single line no newline"},
		{"\nblank first line", ""},
	}
	for _, tc := range cases {
		got := firstLine([]byte(tc.input))
		if got != tc.want {
			t.Errorf("firstLine(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestIsTTY_PipedStdin(t *testing.T) {
	// When running under `go test`, stdin is never a TTY
	if isTTY() {
		t.Skip("test environment has a real TTY attached")
	}
}

func TestFileExists(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "exists")
	os.WriteFile(existing, []byte("x"), 0o644)

	if !fileExists(existing) {
		t.Error("expected true for existing file")
	}
	if fileExists(filepath.Join(dir, "nope")) {
		t.Error("expected false for non-existent file")
	}
	if fileExists(dir) {
		t.Error("expected false for directory")
	}
}
