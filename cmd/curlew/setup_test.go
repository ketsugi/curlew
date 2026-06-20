package main

import (
	"os"
	"testing"
)

// withStdin replaces os.Stdin with a pipe carrying the given content (closed
// immediately so reads see EOF after it), runs fn, and restores os.Stdin.
func withStdin(t *testing.T, content string, fn func()) {
	t.Helper()
	orig := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdin = r
	if content != "" {
		w.WriteString(content)
	}
	w.Close() // EOF after content
	defer func() { os.Stdin = orig; r.Close() }()
	fn()
}

func TestPromptYes_EOFDeclines(t *testing.T) {
	withStdin(t, "", func() {
		if promptYes() {
			t.Error("EOF/empty stdin must decline (return false), got true")
		}
	})
}

func TestPromptYes_BareEnterAccepts(t *testing.T) {
	withStdin(t, "\n", func() {
		if !promptYes() {
			t.Error("bare Enter should accept the default (true), got false")
		}
	})
}

func TestPromptYes_ExplicitNo(t *testing.T) {
	for _, in := range []string{"n\n", "N\n", "no\n", "No\n"} {
		withStdin(t, in, func() {
			if promptYes() {
				t.Errorf("input %q should decline", in)
			}
		})
	}
}

func TestPromptYes_ExplicitYes(t *testing.T) {
	for _, in := range []string{"y\n", "Y\n", "yes\n"} {
		withStdin(t, in, func() {
			if !promptYes() {
				t.Errorf("input %q should accept", in)
			}
		})
	}
}

func TestInteractive_SkipCheckEnv(t *testing.T) {
	t.Setenv("CURLEW_SKIP_TTY_CHECK", "1")
	if !interactive() {
		t.Error("CURLEW_SKIP_TTY_CHECK=1 should force interactive=true")
	}
}

func TestInteractive_PipedStdinNotTTY(t *testing.T) {
	t.Setenv("CURLEW_SKIP_TTY_CHECK", "")
	withStdin(t, "", func() {
		if interactive() {
			t.Error("a pipe is not a terminal — interactive should be false")
		}
	})
}
