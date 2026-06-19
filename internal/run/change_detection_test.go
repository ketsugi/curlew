package run

import (
	"testing"

	"github.com/ketsugi/curlew/internal/config"
	"github.com/ketsugi/curlew/internal/ledger"
)

func TestCheckForChanges_NeverSeen(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", stateDir)

	result := checkForChanges("https://example.com/install.sh", "abc123")
	if result != changeNew {
		t.Errorf("expected changeNew, got %v", result)
	}
}

func TestCheckForChanges_Unchanged(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", stateDir)

	l, _ := ledger.New(config.LedgerDir())
	l.Record(ledger.Entry{URL: "https://example.com/install.sh", SHA256: "abc123"})

	result := checkForChanges("https://example.com/install.sh", "abc123")
	if result != changeUnchanged {
		t.Errorf("expected changeUnchanged, got %v", result)
	}
}

func TestCheckForChanges_Changed(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", stateDir)

	l, _ := ledger.New(config.LedgerDir())
	l.Record(ledger.Entry{URL: "https://example.com/install.sh", SHA256: "old-hash"})

	result := checkForChanges("https://example.com/install.sh", "new-hash")
	if result != changeModified {
		t.Errorf("expected changeModified, got %v", result)
	}
}

func TestCheckForChanges_LocalFileSkipped(t *testing.T) {
	result := checkForChanges("/tmp/local-script.sh", "abc123")
	if result != changeNew {
		t.Errorf("expected changeNew for local file, got %v", result)
	}
}
