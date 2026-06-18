package run

import (
	"bytes"
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
