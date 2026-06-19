package ai

import (
	"fmt"
	"strings"
)

// ResolveCommand determines the AI analysis backend command.
// Returns the command as a string slice suitable for exec.
//
// Precedence:
//  1. aiCmd — raw command override (split on whitespace)
//  2. ai preset ("claude" or "ollama") + model
//
// For the "claude" preset, claudeCmd overrides the binary name.
func ResolveCommand(ai, model, aiCmd, claudeCmd string) ([]string, error) {
	if aiCmd != "" {
		parts := strings.Fields(aiCmd)
		if len(parts) == 0 {
			return nil, fmt.Errorf("ai_cmd is set but empty after splitting")
		}
		return parts, nil
	}

	if ai == "" {
		ai = "claude"
	}

	switch ai {
	case "claude":
		bin := claudeCmd
		if bin == "" {
			bin = "claude"
		}
		if model == "" {
			model = "sonnet"
		}
		return []string{bin, "--model", model, "--print"}, nil

	case "ollama":
		if model == "" {
			return nil, fmt.Errorf("ai=ollama requires a model (e.g. model = \"llama3.2\")")
		}
		// --nowordwrap stops ollama from emitting cursor-control reflow
		// sequences (ESC[K / ESC[nD) that glow renders as literal garbage.
		return []string{"ollama", "run", "--nowordwrap", model}, nil

	default:
		return nil, fmt.Errorf("Unknown AI backend: %s (supported: claude, ollama; or set ai_cmd)", ai)
	}
}
