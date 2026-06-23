package run

import (
	"fmt"
	"io"
	"net/http"
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
	n, err := fetchInto(target, dst)
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("File is empty")
	}
	return nil
}

// fetchInto copies the target (local file or remote URL) into dst and returns
// the number of bytes written. Emptiness is checked once by the caller.
func fetchInto(target string, dst *os.File) (int64, error) {
	if fileExists(target) {
		info("Reading local file: %s", target)
		src, err := os.Open(target)
		if err != nil {
			return 0, err
		}
		defer src.Close()
		return io.Copy(dst, src)
	}

	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		info("Downloading: %s", target)
		client := &http.Client{Timeout: downloadTimeout}
		resp, err := client.Get(target)
		if err != nil {
			return 0, fmt.Errorf("Failed to download: %s", target)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return 0, fmt.Errorf("Failed to download: %s (HTTP %d)", target, resp.StatusCode)
		}
		n, err := io.Copy(dst, io.LimitReader(resp.Body, maxDownloadBytes+1))
		if err != nil {
			return 0, err
		}
		if n > maxDownloadBytes {
			return 0, fmt.Errorf("Download exceeds %d bytes — refusing to proceed", maxDownloadBytes)
		}
		return n, nil
	}

	return 0, fmt.Errorf("Not a valid URL or local file: %s", target)
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
