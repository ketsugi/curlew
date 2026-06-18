package validate

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
)

// Non-text/* types that are still acceptable script content.
// The bash version accepted: text/* | application/x-shellscript | application/javascript
var allowedMIMETypes = map[string]bool{
	"application/x-shellscript": true,
	"application/javascript":    true,
}

// MIMEType detects the MIME type of the file at path.
// Returns the detected type and nil if it's a valid text-based script type.
// Returns the detected type and an error if it's not acceptable.
func MIMEType(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil {
		return "", err
	}

	detected := http.DetectContentType(buf[:n])
	// http.DetectContentType may return params (e.g. "text/plain; charset=utf-8")
	mime := strings.SplitN(detected, ";", 2)[0]
	mime = strings.TrimSpace(mime)

	if strings.HasPrefix(mime, "text/") || allowedMIMETypes[mime] {
		return mime, nil
	}

	return mime, fmt.Errorf("not a text-based script (detected: %s)", mime)
}

// HasNullBytes returns true if the file contains any null bytes.
func HasNullBytes(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanBytes)
	for scanner.Scan() {
		if scanner.Bytes()[0] == 0 {
			return true, nil
		}
	}
	return false, scanner.Err()
}

var injectionRe = regexp.MustCompile(`(?i)(ignore (all )?(previous|above|prior) instructions|you are now|disregard (the |all )?(above|previous|prior)|forget your (instructions|prompt)|new instructions:)`)

// HasInjectionPatterns returns true if the file contains potential LLM prompt
// injection patterns.
func HasInjectionPatterns(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	return injectionRe.Match(data), nil
}

// Known benign flags per interpreter.
var benignFlags = map[string]map[string]bool{
	"bash":    {"-w": true, "-u": true, "-e": true, "-x": true, "-O": true, "-OO": true},
	"sh":      {"-w": true, "-u": true, "-e": true, "-x": true, "-O": true, "-OO": true},
	"perl":    {"-w": true, "-u": true, "-e": true, "-x": true, "-O": true, "-OO": true},
	"python":  {"-w": true, "-u": true, "-e": true, "-x": true, "-O": true, "-OO": true},
	"python3": {"-w": true, "-u": true, "-e": true, "-x": true, "-O": true, "-OO": true},
	"ruby":    {"-w": true, "-u": true, "-e": true, "-x": true, "-O": true, "-OO": true},
	"node":    {"-w": true, "-u": true, "-e": true, "-x": true, "-O": true, "-OO": true},
}

// ValidateShebang checks whether a shebang line is safe to execute.
// Returns nil if safe, or an error describing the rejection reason.
func ValidateShebang(line string) error {
	if !strings.HasPrefix(line, "#!") {
		return nil
	}

	interpStr := strings.TrimSpace(line[2:])
	parts := strings.Fields(interpStr)

	if len(parts) <= 1 {
		return nil
	}

	basename := parts[0]
	if idx := strings.LastIndex(basename, "/"); idx >= 0 {
		basename = basename[idx+1:]
	}

	switch basename {
	case "env":
		args := parts[1:]
		if len(args) > 0 && args[0] == "-S" {
			args = args[1:]
		}
		if len(args) < 1 {
			return fmt.Errorf("Refusing degenerate env shebang: %s", interpStr)
		}
		if len(args) > 1 {
			return fmt.Errorf("Refusing complex env shebang: %s", interpStr)
		}
		return nil

	case "bash", "sh", "perl", "python", "python3", "ruby", "node":
		if len(parts) == 2 {
			flag := parts[1]
			if benignFlags[basename][flag] {
				return nil
			}
			return fmt.Errorf("Refusing shebang flag: %s", flag)
		}
		return fmt.Errorf("Refusing multi-arg shebang: %s", interpStr)

	default:
		return fmt.Errorf("Refusing multi-arg shebang: %s", interpStr)
	}
}

// GetInterpreter returns the interpreter command from a shebang line.
// If no shebang is present, returns ["bash"].
func GetInterpreter(line string) []string {
	if strings.HasPrefix(line, "#!") {
		interpStr := strings.TrimSpace(line[2:])
		if fields := strings.Fields(interpStr); len(fields) > 0 {
			return fields
		}
		// A bare "#!" (no interpreter token) falls back to bash, matching the
		// no-shebang case — otherwise the caller would try to exec the script
		// file directly.
	}
	return []string{"bash"}
}
