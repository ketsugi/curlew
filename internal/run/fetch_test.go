package run

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fetchToString runs fetch into a fresh temp file and returns the written content.
func fetchToString(t *testing.T, target string) (string, error) {
	t.Helper()
	tmp, err := os.CreateTemp(t.TempDir(), "curlew-fetch-*")
	if err != nil {
		t.Fatal(err)
	}
	defer tmp.Close()
	if ferr := fetch(target, tmp); ferr != nil {
		return "", ferr
	}
	data, err := os.ReadFile(tmp.Name())
	if err != nil {
		t.Fatal(err)
	}
	return string(data), nil
}

func TestFetch_HTTPSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "#!/bin/bash\necho hi\n")
	}))
	defer srv.Close()

	got, err := fetchToString(t, srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "echo hi") {
		t.Errorf("body not written, got %q", got)
	}
}

func TestFetch_HTTPNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := fetchToString(t, srv.URL)
	if err == nil {
		t.Fatal("expected error for HTTP 404")
	}
	if !strings.Contains(err.Error(), "Failed to download") {
		t.Errorf("expected 'Failed to download', got %v", err)
	}
}

func TestFetch_FollowsRedirect(t *testing.T) {
	final := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "redirected-content")
	}))
	defer final.Close()
	redir := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, final.URL, http.StatusFound)
	}))
	defer redir.Close()

	got, err := fetchToString(t, redir.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "redirected-content" {
		t.Errorf("redirect not followed, got %q", got)
	}
}

func TestFetch_SizeCap(t *testing.T) {
	old := maxDownloadBytes
	maxDownloadBytes = 16
	defer func() { maxDownloadBytes = old }()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(strings.Repeat("A", 100)))
	}))
	defer srv.Close()

	_, err := fetchToString(t, srv.URL)
	if err == nil {
		t.Fatal("expected size-cap error")
	}
	if !strings.Contains(err.Error(), "exceeds") {
		t.Errorf("expected size-cap error, got %v", err)
	}
}

func TestFetch_HTTPEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// empty body
	}))
	defer srv.Close()

	_, err := fetchToString(t, srv.URL)
	if err == nil {
		t.Fatal("expected empty-file error")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("expected 'empty', got %v", err)
	}
}

func TestFetch_UnsupportedScheme(t *testing.T) {
	for _, scheme := range []string{"gopher://example.com/x", "ftp://example.com/x"} {
		_, err := fetchToString(t, scheme)
		if err == nil {
			t.Fatalf("%s: expected error for unsupported scheme", scheme)
		}
		if !strings.Contains(err.Error(), "Not a valid URL or local file") {
			t.Errorf("%s: expected 'Not a valid URL or local file', got %v", scheme, err)
		}
	}
}

func TestFetch_LocalFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "script.sh")
	if err := os.WriteFile(p, []byte("#!/bin/sh\necho local\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := fetchToString(t, p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "echo local") {
		t.Errorf("local file not read, got %q", got)
	}
}

func TestFetch_LocalEmpty(t *testing.T) {
	p := filepath.Join(t.TempDir(), "empty.sh")
	if err := os.WriteFile(p, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := fetchToString(t, p)
	if err == nil {
		t.Fatal("expected empty-file error")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("expected 'empty', got %v", err)
	}
}
