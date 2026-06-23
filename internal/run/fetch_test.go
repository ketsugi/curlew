package run

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsDowngrade(t *testing.T) {
	cases := []struct {
		from, to string
		want     bool
	}{
		{"https://a.com/x", "http://a.com/x", true},
		{"https://a.com/x", "https://b.com/x", false},
		{"http://a.com/x", "http://b.com/x", false},
		{"http://a.com/x", "https://a.com/x", false},
	}
	for _, c := range cases {
		from, _ := url.Parse(c.from)
		to, _ := url.Parse(c.to)
		if got := isDowngrade(from, to); got != c.want {
			t.Errorf("isDowngrade(%s, %s) = %v, want %v", c.from, c.to, got, c.want)
		}
	}
}

func req(rawurl string) *http.Request {
	u, _ := url.Parse(rawurl)
	return &http.Request{URL: u}
}

func TestCheckRedirect_AllowsNormalRedirect(t *testing.T) {
	via := []*http.Request{req("https://a.com/x")}
	if err := checkRedirect(req("https://b.com/x"), via); err != nil {
		t.Errorf("expected nil for a normal redirect, got %v", err)
	}
}

func TestCheckRedirect_AllowsDowngradeButWarns(t *testing.T) {
	// A downgrade is surfaced via a warning but not blocked.
	via := []*http.Request{req("https://a.com/x")}
	if err := checkRedirect(req("http://a.com/x"), via); err != nil {
		t.Errorf("downgrade should not be blocked, got %v", err)
	}
}

func TestCheckRedirect_StopsAfterTenRedirects(t *testing.T) {
	via := make([]*http.Request, 10)
	for i := range via {
		via[i] = req("http://a.com/x")
	}
	if err := checkRedirect(req("http://a.com/x"), via); err == nil {
		t.Error("expected error after 10 redirects")
	}
}

// requireNetwork skips the test if we can't bind a local port (e.g. sandbox).
func requireNetwork(t *testing.T) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("cannot bind local port: %v", err)
	}
	ln.Close()
}

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
	requireNetwork(t)
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
	requireNetwork(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := fetchToString(t, srv.URL)
	if err == nil {
		t.Fatal("expected error for HTTP 404")
	}
	if !strings.Contains(err.Error(), "failed to download") {
		t.Errorf("expected 'failed to download', got %v", err)
	}
}

func TestFetch_FollowsRedirect(t *testing.T) {
	requireNetwork(t)
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
	requireNetwork(t)
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
	requireNetwork(t)
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
		if !strings.Contains(err.Error(), "not a valid URL or local file") {
			t.Errorf("%s: expected 'not a valid URL or local file', got %v", scheme, err)
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
