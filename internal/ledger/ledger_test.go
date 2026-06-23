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

func TestRecord_UpdatesStoredScript(t *testing.T) {
	dir := t.TempDir()
	l, _ := New(dir)

	url := "https://example.com/install.sh"
	v1 := []byte("#!/bin/bash\necho v1\n")
	v2 := []byte("#!/bin/bash\necho v2\n")

	l.Record(Entry{URL: url, SHA256: "v1", Script: v1})
	l.Record(Entry{URL: url, SHA256: "v2", Script: v2})

	entries, _ := l.List()
	got, err := l.GetScript(entries[0])
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(v2) {
		t.Errorf("expected stored script updated to v2, got %q", got)
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
	if e.RunCount != 0 {
		t.Errorf("expected RunCount=0 (not executed), got %d", e.RunCount)
	}
}

func TestRecord_UpdatesHashOnSameURL(t *testing.T) {
	dir := t.TempDir()
	l, _ := New(dir)

	l.Record(Entry{URL: "https://example.com/install.sh", SHA256: "v1"})
	l.Record(Entry{URL: "https://example.com/install.sh", SHA256: "v2"})

	entries, _ := l.List()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry (same URL), got %d", len(entries))
	}
	if entries[0].SHA256 != "v2" {
		t.Errorf("expected SHA256 updated to v2, got %q", entries[0].SHA256)
	}
	if entries[0].RunCount != 0 {
		t.Errorf("expected RunCount=0 (Record doesn't increment), got %d", entries[0].RunCount)
	}
}

func TestMarkExecuted(t *testing.T) {
	dir := t.TempDir()
	l, _ := New(dir)

	l.Record(Entry{URL: "https://example.com/install.sh", SHA256: "abc"})
	l.MarkExecuted("https://example.com/install.sh")

	entries, _ := l.List()
	if !entries[0].Executed {
		t.Error("expected Executed=true after MarkExecuted")
	}
	if entries[0].RunCount != 1 {
		t.Errorf("expected RunCount=1, got %d", entries[0].RunCount)
	}
}

func TestMarkExecuted_IncrementsOnRepeat(t *testing.T) {
	dir := t.TempDir()
	l, _ := New(dir)

	l.Record(Entry{URL: "https://example.com/install.sh", SHA256: "abc"})
	l.MarkExecuted("https://example.com/install.sh")
	l.MarkExecuted("https://example.com/install.sh")

	entries, _ := l.List()
	if entries[0].RunCount != 2 {
		t.Errorf("expected RunCount=2, got %d", entries[0].RunCount)
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

// --- Schema versioning ---

func TestRecord_StampsSchemaVersion(t *testing.T) {
	dir := t.TempDir()
	l, _ := New(dir)

	l.Record(Entry{URL: "https://example.com/x.sh", SHA256: "abc"})

	e, err := l.Lookup("https://example.com/x.sh")
	if err != nil {
		t.Fatal(err)
	}
	if e.SchemaVersion != schemaVersion {
		t.Errorf("expected schema_version %d, got %d", schemaVersion, e.SchemaVersion)
	}
}

func TestReadMeta_SkipsNewerSchemaVersion(t *testing.T) {
	dir := t.TempDir()
	l, _ := New(dir)

	// Hand-write an entry claiming a future schema version. A binary that only
	// understands the current version must not misparse it.
	url := "https://example.com/future.sh"
	entryDir := filepath.Join(dir, urlHash(url))
	if err := os.MkdirAll(entryDir, 0o755); err != nil {
		t.Fatal(err)
	}
	meta := `{"schema_version": 999, "url": "https://example.com/future.sh", "sha256": "z"}`
	if err := os.WriteFile(filepath.Join(entryDir, "metadata.json"), []byte(meta), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, err := l.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("expected newer-version entry to be skipped, got %d entries", len(entries))
	}

	e, err := l.Lookup(url)
	if e != nil {
		t.Errorf("expected nil entry for newer schema version, got %+v (err=%v)", e, err)
	}
}

func TestReadMeta_AcceptsLegacyUnversioned(t *testing.T) {
	dir := t.TempDir()
	l, _ := New(dir)

	// A pre-versioning entry has no schema_version field (defaults to 0).
	// Its shape is identical to the current format, so it must still load.
	url := "https://example.com/legacy.sh"
	entryDir := filepath.Join(dir, urlHash(url))
	if err := os.MkdirAll(entryDir, 0o755); err != nil {
		t.Fatal(err)
	}
	meta := `{"url": "https://example.com/legacy.sh", "sha256": "abc"}`
	if err := os.WriteFile(filepath.Join(entryDir, "metadata.json"), []byte(meta), 0o644); err != nil {
		t.Fatal(err)
	}

	e, err := l.Lookup(url)
	if err != nil {
		t.Fatal(err)
	}
	if e == nil || e.SHA256 != "abc" {
		t.Errorf("expected legacy entry to load, got %+v", e)
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
