package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config holds curlew's runtime configuration.
// Precedence: CLI flags > env vars > config file > defaults.
type Config struct {
	AI        string `toml:"ai"`
	Model     string `toml:"model"`
	AICmd     string `toml:"ai_cmd"`
	Threshold int    `toml:"threshold"`
}

// Defaults returns the built-in default configuration.
func Defaults() Config {
	return Config{
		AI:        "claude",
		Model:     "sonnet",
		Threshold: 20,
	}
}

// Load reads the config file and layers env vars on top.
// Missing file is not an error — defaults are returned.
func Load() (Config, error) {
	cfg := Defaults()

	path := filePath()
	if path != "" {
		if _, err := os.Stat(path); err == nil {
			if _, err := toml.DecodeFile(path, &cfg); err != nil {
				return cfg, err
			}
		}
	}

	// Env vars override config file
	if v := os.Getenv("CURLEW_AI"); v != "" {
		cfg.AI = v
	}
	if v := os.Getenv("CURLEW_MODEL"); v != "" {
		cfg.Model = v
	}
	if v := os.Getenv("CURLEW_AI_CMD"); v != "" {
		cfg.AICmd = v
	}

	return cfg, nil
}

// FilePath returns the resolved config file path (for --init-config).
func FilePath() string {
	return filePath()
}

func filePath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "curlew", "config.toml")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "curlew", "config.toml")
}

// Template returns a commented config file template.
const Template = `# curlew configuration
# Place at: ~/.config/curlew/config.toml (or $XDG_CONFIG_HOME/curlew/config.toml)
#
# Precedence: CLI flags > env vars > this file > built-in defaults.

# AI backend preset: "claude" or "ollama"
# ai = "claude"

# Model name passed to the preset
# model = "sonnet"

# Raw command override (wins over ai/model preset).
# The command receives the analysis prompt on stdin and writes markdown to stdout.
# ai_cmd = ""

# Auto-suggest AI analysis for scripts longer than this many lines
# threshold = 20
`
