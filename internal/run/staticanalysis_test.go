package run

import (
	"os"
	"strings"
	"testing"
)

// captureStderr runs fn with os.Stderr redirected to a pipe and returns what
// was written.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	fn()
	w.Close()
	os.Stderr = orig

	var buf [8192]byte
	n, _ := r.Read(buf[:])
	r.Close()
	return string(buf[:n])
}

func TestReportStaticAnalysis_Summary(t *testing.T) {
	script := []byte("#!/bin/bash\ncurl https://a.com\ncurl https://b.com\n")
	out := captureStderr(t, func() { reportStaticAnalysis(script) })

	if !strings.Contains(out, "Static analysis:") {
		t.Errorf("expected summary line, got:\n%s", out)
	}
	// Two curls → "2 Network calls" (pluralized)
	if !strings.Contains(out, "2 Network calls") {
		t.Errorf("expected pluralized network count, got:\n%s", out)
	}
}

func TestReportStaticAnalysis_ExpandsNotable(t *testing.T) {
	script := []byte("#!/bin/bash\nsudo chmod 777 /etc/passwd\n")
	out := captureStderr(t, func() { reportStaticAnalysis(script) })

	if !strings.Contains(out, "Privilege escalation") {
		t.Errorf("expected priv-esc category, got:\n%s", out)
	}
	// Expanded breakdown includes the line + command detail.
	if !strings.Contains(out, "L2:") {
		t.Errorf("expected expanded per-finding line for notable category, got:\n%s", out)
	}
}

func TestReportStaticAnalysis_RoutineNotExpanded(t *testing.T) {
	// Only routine categories (network/url) — should stay a one-liner with no
	// per-finding expansion (no "L<n>:" detail lines).
	script := []byte("#!/bin/bash\ncurl https://example.com/x\n")
	out := captureStderr(t, func() { reportStaticAnalysis(script) })

	if strings.Contains(out, "L2:") {
		t.Errorf("routine findings should not expand, got:\n%s", out)
	}
}

func TestReportStaticAnalysis_NoFindingsSilent(t *testing.T) {
	script := []byte("#!/bin/bash\necho hello\nx=1\n")
	out := captureStderr(t, func() { reportStaticAnalysis(script) })

	if strings.Contains(out, "Static analysis") {
		t.Errorf("expected no output for a plain script, got:\n%s", out)
	}
}

func TestReportStaticAnalysis_ParseErrorSilent(t *testing.T) {
	// Unparseable shell — must not panic or print, just skip.
	script := []byte("#!/bin/bash\necho \"unterminated\n")
	out := captureStderr(t, func() { reportStaticAnalysis(script) })

	if strings.Contains(out, "Static analysis") {
		t.Errorf("expected silence on parse error, got:\n%s", out)
	}
}

func TestSummaryLine_SingularPlural(t *testing.T) {
	// One network call stays singular; the renderer pluralizes only at n>1.
	script := []byte("#!/bin/bash\ncurl https://only.example.com\n")
	out := captureStderr(t, func() { reportStaticAnalysis(script) })
	if !strings.Contains(out, "1 Network call,") && !strings.Contains(out, "1 Network call\n") {
		// network + its URL both present; just assert singular "call" not "calls"
		if strings.Contains(out, "Network calls") {
			t.Errorf("single call should be singular, got:\n%s", out)
		}
	}
}
