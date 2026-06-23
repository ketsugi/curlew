package run

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/ketsugi/curlew/internal/config"
	"github.com/ketsugi/curlew/internal/homograph"
	"github.com/ketsugi/curlew/internal/ledger"
	"github.com/ketsugi/curlew/internal/validate"
)

// Options holds the runtime configuration for an execution flow.
type Options struct {
	Target       string
	ForceAnalyze bool
	SkipTTY      bool // CURLEW_SKIP_TTY_CHECK
	Config       config.Config
}

// Execute runs the full curlew interactive flow:
// download → validate → inspect → analyze → confirm → execute.
func Execute(opts Options) error {
	if !opts.SkipTTY && !isTTY() {
		return fmt.Errorf("curlew requires an interactive terminal")
	}

	// --- Check for homograph attacks in URL hostnames ---
	if warnings := homograph.CheckURL(opts.Target); len(warnings) > 0 {
		warn("Hostname contains suspicious characters (possible homograph attack):")
		for _, w := range warnings {
			if w.IsPunycode {
				warn("  Punycode internationalized domain detected")
			} else {
				warn("  U+%04X (%s) — looks like %q", w.Rune, w.Name, w.LooksLike)
			}
		}
		fmt.Fprintln(os.Stderr)
	}

	// --- Download or copy local file ---
	tmp, err := os.CreateTemp("", "curlew.*")
	if err != nil {
		return err
	}
	tmpfile := tmp.Name()
	defer os.Remove(tmpfile)
	stopCleanup := cleanupOnInterrupt(tmpfile)
	defer stopCleanup()

	if err := fetch(opts.Target, tmp); err != nil {
		tmp.Close()
		return err
	}
	tmp.Close()

	// --- Step 1: Validate ---
	mime, err := validate.MIMEType(tmpfile)
	if err != nil {
		return fmt.Errorf("not a text-based script (detected: %s); refusing to proceed", mime)
	}
	info("File type: %s", mime)

	hasNull, err := validate.HasNullBytes(tmpfile)
	if err != nil {
		return err
	}
	if hasNull {
		return fmt.Errorf("file contains null bytes — likely a binary; refusing to proceed")
	}

	lineCount, err := countLines(tmpfile)
	if err != nil {
		return err
	}
	info("Script is %d lines", lineCount)

	// --- Check for changes against ledger ---
	data, err := os.ReadFile(tmpfile)
	if err != nil {
		return err
	}
	scriptHash := fmt.Sprintf("%x", sha256.Sum256(data))
	changed := checkForChanges(opts.Target, scriptHash)

	// Ensure a ledger entry exists for future change detection
	if isURL(opts.Target) {
		ensureLedgerEntry(opts.Target, data)
	}

	suggestInspect := true
	suggestAnalyze := lineCount > opts.Config.Threshold

	switch changed {
	case changeUnchanged:
		info("Previously vetted (unchanged since last run)")
	case changeModified:
		warn("This script has changed since you last examined it!")
		suggestInspect = true
		suggestAnalyze = true
	}

	// --- Static analysis (structural, always-on, dependency-free) ---
	reportStaticAnalysis(data)

	// --- Step 2: Visual inspection ---
	yes, err := confirm("\033[1;33mOpen script in less for inspection? [Y/n]\033[0m ", suggestInspect)
	if err != nil {
		return err
	}
	if yes {
		pager := os.Getenv("PAGER")
		if pager == "" {
			pager = "less"
		}
		cmd := exec.Command(pager, tmpfile)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
	} else {
		info("Skipping visual inspection.")
	}

	// --- Step 3: AI analysis ---
	doAnalyze := false
	var cErr error
	if suggestAnalyze {
		doAnalyze, cErr = confirm("\033[1;33mRun AI analysis? [Y/n]\033[0m ", true)
	} else {
		doAnalyze, cErr = confirm("\033[1;33mRun AI analysis? [y/N]\033[0m ", false)
	}
	if cErr != nil {
		return cErr
	}

	if doAnalyze {
		if err := runAnalysis(tmpfile, opts.ForceAnalyze, opts.Config, opts.Target); err != nil {
			warn("%s", err)
			warn("Skipping AI analysis.")
		}
	} else {
		info("Skipping AI analysis.")
	}

	// --- Step 4: Confirm execution ---
	fmt.Println()
	yes, err = confirm("\033[1;33mExecute this script? [y/N]\033[0m ", false)
	if err != nil {
		return err
	}
	if !yes {
		info("Aborted.")
		return nil
	}

	// --- Execute ---
	info("Executing...")
	shebang := firstLine(data)

	if err := validate.ValidateShebang(shebang); err != nil {
		return err
	}

	interp := validate.GetInterpreter(shebang)
	args := append(interp, tmpfile)
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	// --- Record to ledger ---
	if isURL(opts.Target) {
		recordToLedger(opts.Target, data)
	}

	return nil
}

func isURL(target string) bool {
	return strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://")
}

func recordToLedger(url string, script []byte) {
	ledgerDir := config.LedgerDir()
	if ledgerDir == "" {
		return
	}
	l, err := ledger.New(ledgerDir)
	if err != nil {
		return
	}
	h := sha256.Sum256(script)
	l.Record(ledger.Entry{
		URL:    url,
		SHA256: hex.EncodeToString(h[:]),
		Script: script,
	})
	l.MarkExecuted(url)
}
