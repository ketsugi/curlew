package run

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/ketsugi/curlew/internal/config"
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
		return fmt.Errorf("Not a text-based script (detected: %s). Refusing to proceed.", mime)
	}
	info("File type: %s", mime)

	hasNull, err := validate.HasNullBytes(tmpfile)
	if err != nil {
		return err
	}
	if hasNull {
		return fmt.Errorf("File contains null bytes — likely a binary. Refusing to proceed.")
	}

	lineCount, err := countLines(tmpfile)
	if err != nil {
		return err
	}
	info("Script is %d lines", lineCount)

	// --- Step 2: Visual inspection ---
	yes, err := confirm("\033[1;33mOpen script in less for inspection? [Y/n]\033[0m ", true)
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
	threshold := opts.Config.Threshold
	doAnalyze := false
	var cErr error
	if lineCount > threshold {
		doAnalyze, cErr = confirm(
			fmt.Sprintf("\033[1;33mScript is longer than %d lines. Run AI analysis? [Y/n]\033[0m ", threshold),
			true,
		)
	} else {
		doAnalyze, cErr = confirm("\033[1;33mRun AI analysis? [y/N]\033[0m ", false)
	}
	if cErr != nil {
		return cErr
	}

	if doAnalyze {
		if err := runAnalysis(tmpfile, opts.ForceAnalyze, opts.Config); err != nil {
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
	data, err := os.ReadFile(tmpfile)
	if err != nil {
		return err
	}
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
	return cmd.Run()
}
