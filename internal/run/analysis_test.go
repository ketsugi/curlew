package run

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ketsugi/curlew/internal/config"
)

// TestRunAnalysis_BackendError verifies that a non-zero exit from the AI
// backend produces a warning on stderr rather than silent empty output.
func TestRunAnalysis_BackendError(t *testing.T) {
	dir := t.TempDir()

	// Mock backend that writes to stderr and exits 1
	mockAI := filepath.Join(dir, "fail-ai")
	if err := os.WriteFile(mockAI, []byte("#!/bin/sh\necho 'backend error: connection refused' >&2\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Script to analyze
	script := filepath.Join(dir, "script.sh")
	if err := os.WriteFile(script, []byte("#!/bin/bash\necho hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Capture stderr
	origStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	cfg := config.Defaults()
	cfg.AICmd = mockAI

	err := runAnalysis(script, false, cfg)

	w.Close()
	os.Stderr = origStderr

	var buf [4096]byte
	n, _ := r.Read(buf[:])
	output := string(buf[:n])

	if err != nil {
		t.Fatalf("runAnalysis should return nil (advisory), got: %v", err)
	}

	if !strings.Contains(output, "AI backend exited with an error") {
		t.Errorf("expected warning about backend error in stderr, got:\n%s", output)
	}
}

// TestRunAnalysis_BackendSuccess verifies no spurious warning on success.
func TestRunAnalysis_BackendSuccess(t *testing.T) {
	dir := t.TempDir()

	mockAI := filepath.Join(dir, "ok-ai")
	if err := os.WriteFile(mockAI, []byte("#!/bin/sh\necho 'Analysis: looks fine'\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	script := filepath.Join(dir, "script.sh")
	if err := os.WriteFile(script, []byte("#!/bin/bash\necho hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	origStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	cfg := config.Defaults()
	cfg.AICmd = mockAI

	err := runAnalysis(script, false, cfg)

	w.Close()
	os.Stderr = origStderr

	var buf [4096]byte
	n, _ := r.Read(buf[:])
	output := string(buf[:n])

	if err != nil {
		t.Fatalf("runAnalysis should return nil, got: %v", err)
	}

	if strings.Contains(output, "AI backend exited with an error") {
		t.Errorf("unexpected error warning for successful backend:\n%s", output)
	}
}
