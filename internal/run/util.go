package run

import (
	"fmt"
	"os"
	"strings"
)

func info(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "\033[1;34m==>\033[0m "+format+"\n", args...)
}

func warn(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "\033[1;33mwarning:\033[0m "+format+"\n", args...)
}

func isTTY() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func fileExists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && !fi.IsDir()
}

func countLines(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	if len(data) == 0 {
		return 0, nil
	}
	count := strings.Count(string(data), "\n")
	if data[len(data)-1] != '\n' {
		count++
	}
	return count, nil
}

func firstLine(data []byte) string {
	for i, b := range data {
		if b == '\n' {
			return string(data[:i])
		}
	}
	return string(data)
}
