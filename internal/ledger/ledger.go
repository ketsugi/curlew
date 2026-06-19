package ledger

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Entry represents a single recorded script execution.
type Entry struct {
	URL      string    `json:"url"`
	SHA256   string    `json:"sha256"`
	FirstRun time.Time `json:"first_run"`
	LastRun  time.Time `json:"last_run"`
	RunCount int       `json:"run_count"`
	Script   []byte    `json:"-"`

	// dirName is the hash-based directory name (not serialized)
	dirName string
}

// Ledger manages the persistent record of vetted scripts.
type Ledger struct {
	dir string
}

// New creates a Ledger rooted at dir, creating the directory if needed.
func New(dir string) (*Ledger, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Ledger{dir: dir}, nil
}

// Record adds or updates a ledger entry for the given script.
// If an entry already exists for the same URL, it updates the SHA256,
// LastRun, and RunCount while preserving FirstRun.
func (l *Ledger) Record(e Entry) error {
	hash := urlHash(e.URL)
	entryDir := filepath.Join(l.dir, hash)

	if err := os.MkdirAll(entryDir, 0o755); err != nil {
		return err
	}

	metaPath := filepath.Join(entryDir, "metadata.json")
	now := time.Now().Truncate(time.Millisecond)

	existing, err := l.readMeta(metaPath)
	if err == nil && existing != nil {
		existing.SHA256 = e.SHA256
		existing.LastRun = now
		existing.RunCount++
		e = *existing
	} else {
		e.FirstRun = now
		e.LastRun = now
		e.RunCount = 1
	}

	if err := l.writeMeta(metaPath, &e); err != nil {
		return err
	}

	if len(e.Script) > 0 {
		scriptPath := filepath.Join(entryDir, "script")
		if err := atomicWrite(scriptPath, e.Script); err != nil {
			return err
		}
	}

	return nil
}

// Lookup finds the ledger entry for a URL, or returns nil if not found.
func (l *Ledger) Lookup(url string) (*Entry, error) {
	hash := urlHash(url)
	metaPath := filepath.Join(l.dir, hash, "metadata.json")

	e, err := l.readMeta(metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	e.dirName = hash
	return e, nil
}

// List returns all ledger entries.
func (l *Ledger) List() ([]Entry, error) {
	dirs, err := os.ReadDir(l.dir)
	if err != nil {
		return nil, err
	}

	var entries []Entry
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		metaPath := filepath.Join(l.dir, d.Name(), "metadata.json")
		e, err := l.readMeta(metaPath)
		if err != nil {
			continue
		}
		e.dirName = d.Name()
		entries = append(entries, *e)
	}
	return entries, nil
}

// GetScript returns the stored script content for an entry.
// Returns an error if no script was stored.
func (l *Ledger) GetScript(e Entry) ([]byte, error) {
	hash := urlHash(e.URL)
	scriptPath := filepath.Join(l.dir, hash, "script")
	data, err := os.ReadFile(scriptPath)
	if err != nil {
		return nil, fmt.Errorf("no script stored for %s", e.URL)
	}
	return data, nil
}

func (l *Ledger) readMeta(path string) (*Entry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var e Entry
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, err
	}
	return &e, nil
}

func (l *Ledger) writeMeta(path string, e *Entry) error {
	data, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(path, data)
}

// atomicWrite writes data to path via a temp file + rename.
func atomicWrite(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// urlHash returns a stable, filesystem-safe hash of a URL.
// Normalizes by stripping trailing slashes before hashing.
func urlHash(rawURL string) string {
	normalized := strings.TrimRight(rawURL, "/")
	h := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(h[:6])
}
