package run

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

const (
	defaultMaxDownloadBytes = 25 << 20 // 25 MiB
	downloadTimeout         = 30 * time.Second
)

// maxDownloadBytes is a var (not const) so tests can shrink it.
var maxDownloadBytes int64 = defaultMaxDownloadBytes

// fetch writes the target (local file or remote URL) into dst.
func fetch(target string, dst *os.File) error {
	if fileExists(target) {
		info("Reading local file: %s", target)
		src, err := os.Open(target)
		if err != nil {
			return err
		}
		defer src.Close()
		n, err := io.Copy(dst, src)
		if err != nil {
			return err
		}
		if n == 0 {
			return fmt.Errorf("File is empty")
		}
		return nil
	}

	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		info("Downloading: %s", target)
		if strings.HasPrefix(target, "http://") {
			warn("Downloading over plaintext HTTP — the script can be tampered with in transit: %s", target)
		}
		client := &http.Client{
			Timeout: downloadTimeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("stopped after 10 redirects")
				}
				if prev := via[len(via)-1]; isDowngrade(prev.URL, req.URL) {
					warn("Redirect downgrades HTTPS → HTTP: %s → %s", prev.URL, req.URL)
				}
				return nil
			},
		}
		resp, err := client.Get(target)
		if err != nil {
			return fmt.Errorf("Failed to download: %s", target)
		}
		defer resp.Body.Close()
		if final := resp.Request.URL.String(); final != target {
			info("Redirected to: %s", final)
		}
		if resp.StatusCode != 200 {
			return fmt.Errorf("Failed to download: %s (HTTP %d)", target, resp.StatusCode)
		}
		n, err := io.Copy(dst, io.LimitReader(resp.Body, maxDownloadBytes+1))
		if err != nil {
			return err
		}
		if n > maxDownloadBytes {
			return fmt.Errorf("Download exceeds %d bytes — refusing to proceed", maxDownloadBytes)
		}
		if n == 0 {
			return fmt.Errorf("File is empty")
		}
		return nil
	}

	return fmt.Errorf("Not a valid URL or local file: %s", target)
}

// isDowngrade reports whether a redirect from -> to drops HTTPS for plaintext
// HTTP — a transport-security regression worth flagging to the user.
func isDowngrade(from, to *url.URL) bool {
	return from.Scheme == "https" && to.Scheme == "http"
}

// cleanupOnInterrupt removes path if the process is interrupted (SIGINT/SIGTERM),
// since deferred cleanup does not run on signal-driven exit. The returned stop
// function deregisters the handler on normal completion.
func cleanupOnInterrupt(path string) (stop func()) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	done := make(chan struct{})
	go func() {
		select {
		case <-sigCh:
			os.Remove(path)
			os.Exit(130)
		case <-done:
		}
	}()
	return func() {
		signal.Stop(sigCh)
		close(done)
	}
}
