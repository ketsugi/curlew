package ledger

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNew_CreatesLedgerDir(t *testing.T) {
	dir := t.TempDir()
	_, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Errorf("ledger dir not created: %v", err)
	}
}

func TestRecord_CreatesEntry(t *testing.T) {
	dir := t.TempDir()
	l, _ := New(dir)

	err := l.Record(Entry{
		URL:    "https://example.com/install.sh",
		SHA256: "abcdef1234567890",
		Script: []byte("#!/bin/bash\necho hi\n"),
	})
	if err != nil {
		t.Fatal(err)
	}

	entries, err := l.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].URL != "https://example.com/install.sh" {
		t.Errorf("expected URL preserved, got %q", entries[0].URL)
	}
	if entries[0].SHA256 != "abcdef1234567890" {
		t.Errorf("expected SHA256 preserved, got %q", entries[0].SHA256)
	}
}

func TestRecord_StoresScript(t *testing.T) {
	dir := t.TempDir()
	l, _ := New(dir)

	content := []byte("#!/bin/bash\necho hello\n")
	l.Record(Entry{
		URL:    "https://example.com/install.sh",
		SHA256: "abc123",
		Script: content,
	})

	entries, _ := l.List()
	got, err := l.GetScript(entries[0])
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(content) {
		t.Errorf("script content mismatch: got %q", got)
	}
}

func TestRecord_SetsTimestamps(t *testing.T) {
	dir := t.TempDir()
	l, _ := New(dir)

	before := time.Now().Truncate(time.Second)
	l.Record(Entry{
		URL:    "https://example.com/install.sh",
		SHA256: "abc123",
	})
	after := time.Now().Add(time.Second)

	entries, _ := l.List()
	e := entries[0]
	if e.FirstRun.Before(before) || e.FirstRun.After(after) {
		t.Errorf("FirstRun %v not between %v and %v", e.FirstRun, before, after)
	}
	if e.LastRun.Before(before) || e.LastRun.After(after) {
		t.Errorf("LastRun %v not between %v and %v", e.LastRun, before, after)
	}
	if e.RunCount != 1 {
		t.Errorf("expected RunCount=1, got %d", e.RunCount)
	}
}

func TestRecord_IncrementsRunCount(t *testing.T) {
	dir := t.TempDir()
	l, _ := New(dir)

	l.Record(Entry{URL: "https://example.com/install.sh", SHA256: "v1"})
	l.Record(Entry{URL: "https://example.com/install.sh", SHA256: "v2"})

	entries, _ := l.List()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry (same URL), got %d", len(entries))
	}
	if entries[0].RunCount != 2 {
		t.Errorf("expected RunCount=2, got %d", entries[0].RunCount)
	}
	if entries[0].SHA256 != "v2" {
		t.Errorf("expected SHA256 updated to v2, got %q", entries[0].SHA256)
	}
}

func TestRecord_PreservesFirstRun(t *testing.T) {
	dir := t.TempDir()
	l, _ := New(dir)

	l.Record(Entry{URL: "https://example.com/install.sh", SHA256: "v1"})
	entries, _ := l.List()
	firstRun := entries[0].FirstRun

	time.Sleep(10 * time.Millisecond)
	l.Record(Entry{URL: "https://example.com/install.sh", SHA256: "v2"})

	entries, _ = l.List()
	if !entries[0].FirstRun.Equal(firstRun) {
		t.Errorf("FirstRun changed from %v to %v", firstRun, entries[0].FirstRun)
	}
	if !entries[0].LastRun.After(firstRun) {
		t.Error("LastRun should be after FirstRun on second record")
	}
}

func TestLookup_ByURL(t *testing.T) {
	dir := t.TempDir()
	l, _ := New(dir)

	l.Record(Entry{URL: "https://example.com/install.sh", SHA256: "abc"})
	l.Record(Entry{URL: "https://other.com/setup.sh", SHA256: "def"})

	e, err := l.Lookup("https://example.com/install.sh")
	if err != nil {
		t.Fatal(err)
	}
	if e == nil {
		t.Fatal("expected entry, got nil")
	}
	if e.SHA256 != "abc" {
		t.Errorf("expected SHA256=abc, got %q", e.SHA256)
	}
}

func TestLookup_NotFound(t *testing.T) {
	dir := t.TempDir()
	l, _ := New(dir)

	e, err := l.Lookup("https://nonexistent.com/x.sh")
	if err != nil {
		t.Fatal(err)
	}
	if e != nil {
		t.Errorf("expected nil for missing URL, got %+v", e)
	}
}

func TestList_Empty(t *testing.T) {
	dir := t.TempDir()
	l, _ := New(dir)

	entries, err := l.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestList_MultipleEntries(t *testing.T) {
	dir := t.TempDir()
	l, _ := New(dir)

	l.Record(Entry{URL: "https://a.com/install.sh", SHA256: "aaa"})
	l.Record(Entry{URL: "https://b.com/setup.sh", SHA256: "bbb"})
	l.Record(Entry{URL: "https://c.com/run.sh", SHA256: "ccc"})

	entries, _ := l.List()
	if len(entries) != 3 {
		t.Errorf("expected 3 entries, got %d", len(entries))
	}
}

func TestURLHash_Deterministic(t *testing.T) {
	h1 := urlHash("https://example.com/install.sh")
	h2 := urlHash("https://example.com/install.sh")
	if h1 != h2 {
		t.Errorf("same URL produced different hashes: %s vs %s", h1, h2)
	}
}

func TestURLHash_NormalizesTrailingSlash(t *testing.T) {
	h1 := urlHash("https://example.com/install.sh")
	h2 := urlHash("https://example.com/install.sh/")
	if h1 != h2 {
		t.Errorf("trailing slash should not affect hash: %s vs %s", h1, h2)
	}
}

func TestURLHash_DifferentURLs(t *testing.T) {
	h1 := urlHash("https://example.com/install.sh")
	h2 := urlHash("https://other.com/install.sh")
	if h1 == h2 {
		t.Error("different URLs should produce different hashes")
	}
}

func TestGetScript_NoScript(t *testing.T) {
	dir := t.TempDir()
	l, _ := New(dir)

	l.Record(Entry{URL: "https://example.com/install.sh", SHA256: "abc"})

	entries, _ := l.List()
	_, err := l.GetScript(entries[0])
	if err == nil {
		t.Error("expected error when no script stored")
	}
}

// --- Analysis cache ---

func TestSaveAnalysis_StoresContent(t *testing.T) {
	dir := t.TempDir()
	l, _ := New(dir)
	l.Record(Entry{URL: "https://example.com/install.sh", SHA256: "abc"})

	err := l.SaveAnalysis("https://example.com/install.sh", Analysis{
		Content: "This script installs foo.",
		Backend: "claude/sonnet",
	})
	if err != nil {
		t.Fatal(err)
	}

	a, err := l.GetAnalysis("https://example.com/install.sh")
	if err != nil {
		t.Fatal(err)
	}
	if a == nil {
		t.Fatal("expected analysis, got nil")
	}
	if a.Content != "This script installs foo." {
		t.Errorf("content mismatch: got %q", a.Content)
	}
	if a.Backend != "claude/sonnet" {
		t.Errorf("expected Backend=claude/sonnet, got %q", a.Backend)
	}
}

func TestSaveAnalysis_StoresTimestamp(t *testing.T) {
	dir := t.TempDir()
	l, _ := New(dir)
	l.Record(Entry{URL: "https://example.com/install.sh", SHA256: "abc"})

	before := time.Now().Truncate(time.Second)
	l.SaveAnalysis("https://example.com/install.sh", Analysis{
		Content: "Safe.",
		Backend: "ollama/llama3",
	})

	a, _ := l.GetAnalysis("https://example.com/install.sh")
	if a.CreatedAt.Before(before) {
		t.Errorf("CreatedAt %v should be after %v", a.CreatedAt, before)
	}
}

func TestGetAnalysis_NotFound(t *testing.T) {
	dir := t.TempDir()
	l, _ := New(dir)

	a, err := l.GetAnalysis("https://nonexistent.com/x.sh")
	if err != nil {
		t.Fatal(err)
	}
	if a != nil {
		t.Errorf("expected nil for missing analysis, got %+v", a)
	}
}

func TestGetAnalysis_NoAnalysisStored(t *testing.T) {
	dir := t.TempDir()
	l, _ := New(dir)
	l.Record(Entry{URL: "https://example.com/install.sh", SHA256: "abc"})

	a, err := l.GetAnalysis("https://example.com/install.sh")
	if err != nil {
		t.Fatal(err)
	}
	if a != nil {
		t.Errorf("expected nil when no analysis saved, got %+v", a)
	}
}

func TestSaveAnalysis_InvalidatedByNewHash(t *testing.T) {
	dir := t.TempDir()
	l, _ := New(dir)
	l.Record(Entry{URL: "https://example.com/install.sh", SHA256: "v1"})
	l.SaveAnalysis("https://example.com/install.sh", Analysis{
		Content: "Old analysis.",
		Backend: "claude/sonnet",
	})

	// Re-record with a different hash (script changed)
	l.Record(Entry{URL: "https://example.com/install.sh", SHA256: "v2"})

	// Analysis should be invalidated
	a, err := l.GetAnalysis("https://example.com/install.sh")
	if err != nil {
		t.Fatal(err)
	}
	if a != nil {
		t.Errorf("expected nil after hash change, got %+v", a)
	}
}

func TestRecord_AtomicWrite(t *testing.T) {
	dir := t.TempDir()
	l, _ := New(dir)

	l.Record(Entry{URL: "https://example.com/install.sh", SHA256: "abc"})

	// No .tmp files should remain after write
	matches, _ := filepath.Glob(filepath.Join(dir, "*", "*.tmp"))
	if len(matches) != 0 {
		t.Errorf("temp files left behind: %v", matches)
	}
}
