package ai

import (
	"fmt"
	"os"
	"strings"
)

// ResolveCommand determines the AI analysis backend command from environment
// variables. Returns the command as a string slice suitable for exec.
//
// Precedence:
//  1. CURLEW_AI_CMD — raw command override (split on whitespace)
//  2. CURLEW_AI preset ("claude" or "ollama") + CURLEW_MODEL
//
// For the "claude" preset, CURLEW_CLAUDE_CMD overrides the binary name.
func ResolveCommand() ([]string, error) {
	if override := os.Getenv("CURLEW_AI_CMD"); override != "" {
		parts := strings.Fields(override)
		if len(parts) == 0 {
			return nil, fmt.Errorf("CURLEW_AI_CMD is set but empty after splitting")
		}
		return parts, nil
	}

	ai := os.Getenv("CURLEW_AI")
	if ai == "" {
		ai = "claude"
	}
	model := os.Getenv("CURLEW_MODEL")

	switch ai {
	case "claude":
		bin := os.Getenv("CURLEW_CLAUDE_CMD")
		if bin == "" {
			bin = "claude"
		}
		if model == "" {
			model = "sonnet"
		}
		return []string{bin, "--model", model, "--print"}, nil

	case "ollama":
		if model == "" {
			return nil, fmt.Errorf("CURLEW_AI=ollama requires CURLEW_MODEL (e.g. CURLEW_MODEL=llama3.2)")
		}
		return []string{"ollama", "run", model}, nil

	default:
		return nil, fmt.Errorf("Unknown CURLEW_AI backend: %s (supported: claude, ollama; or set CURLEW_AI_CMD)", ai)
	}
}
