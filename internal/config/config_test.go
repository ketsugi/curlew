package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaults(t *testing.T) {
	cfg := Defaults()
	if cfg.AI != "claude" {
		t.Errorf("expected ai=claude, got %q", cfg.AI)
	}
	if cfg.Model != "sonnet" {
		t.Errorf("expected model=sonnet, got %q", cfg.Model)
	}
	if cfg.Threshold != 20 {
		t.Errorf("expected threshold=20, got %d", cfg.Threshold)
	}
}

func TestLoad_NoFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CURLEW_AI", "")
	t.Setenv("CURLEW_MODEL", "")
	t.Setenv("CURLEW_AI_CMD", "")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg != Defaults() {
		t.Errorf("expected defaults when no file, got %+v", cfg)
	}
}

func TestLoad_FromFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("CURLEW_AI", "")
	t.Setenv("CURLEW_MODEL", "")
	t.Setenv("CURLEW_AI_CMD", "")

	cfgDir := filepath.Join(dir, "curlew")
	os.MkdirAll(cfgDir, 0o755)
	os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(`
ai = "ollama"
model = "llama3"
threshold = 50
`), 0o644)

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AI != "ollama" {
		t.Errorf("expected ai=ollama, got %q", cfg.AI)
	}
	if cfg.Model != "llama3" {
		t.Errorf("expected model=llama3, got %q", cfg.Model)
	}
	if cfg.Threshold != 50 {
		t.Errorf("expected threshold=50, got %d", cfg.Threshold)
	}
}

func TestLoad_EnvOverridesFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	cfgDir := filepath.Join(dir, "curlew")
	os.MkdirAll(cfgDir, 0o755)
	os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(`
ai = "ollama"
model = "llama3"
`), 0o644)

	t.Setenv("CURLEW_AI", "claude")
	t.Setenv("CURLEW_MODEL", "opus")
	t.Setenv("CURLEW_AI_CMD", "")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AI != "claude" {
		t.Errorf("expected env override ai=claude, got %q", cfg.AI)
	}
	if cfg.Model != "opus" {
		t.Errorf("expected env override model=opus, got %q", cfg.Model)
	}
}

func TestLoad_AICmdOverridesAll(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("CURLEW_AI", "ollama")
	t.Setenv("CURLEW_MODEL", "llama3")
	t.Setenv("CURLEW_AI_CMD", "my-custom-ai --run")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AICmd != "my-custom-ai --run" {
		t.Errorf("expected ai_cmd override, got %q", cfg.AICmd)
	}
}

func TestLoad_ThresholdEnvOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("CURLEW_AI", "")
	t.Setenv("CURLEW_MODEL", "")
	t.Setenv("CURLEW_AI_CMD", "")
	t.Setenv("CURLEW_THRESHOLD", "50")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Threshold != 50 {
		t.Errorf("expected threshold=50, got %d", cfg.Threshold)
	}
}

func TestLoad_ThresholdZeroMeansAlwaysSuggest(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("CURLEW_AI", "")
	t.Setenv("CURLEW_MODEL", "")
	t.Setenv("CURLEW_AI_CMD", "")
	t.Setenv("CURLEW_THRESHOLD", "0")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Threshold != 0 {
		t.Errorf("expected threshold=0, got %d", cfg.Threshold)
	}
}

func TestLoad_ThresholdInvalidIgnored(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("CURLEW_AI", "")
	t.Setenv("CURLEW_MODEL", "")
	t.Setenv("CURLEW_AI_CMD", "")
	t.Setenv("CURLEW_THRESHOLD", "notanumber")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Threshold != 20 {
		t.Errorf("expected default threshold=20 when env invalid, got %d", cfg.Threshold)
	}
}

func TestLedgerDir_XDGStateHome(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/tmp/test-state")
	got := LedgerDir()
	if got != "/tmp/test-state/curlew/ledger" {
		t.Errorf("expected /tmp/test-state/curlew/ledger, got %q", got)
	}
}

func TestLedgerDir_DefaultsToHome(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "")
	got := LedgerDir()
	if got == "" {
		t.Skip("no home directory available")
	}
	if !strings.Contains(got, ".local/state/curlew/ledger") {
		t.Errorf("expected path containing .local/state/curlew/ledger, got %q", got)
	}
}

func TestLoad_InvalidTOML(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("CURLEW_AI", "")
	t.Setenv("CURLEW_MODEL", "")
	t.Setenv("CURLEW_AI_CMD", "")

	cfgDir := filepath.Join(dir, "curlew")
	os.MkdirAll(cfgDir, 0o755)
	os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(`not valid [[[ toml`), 0o644)

	_, err := Load()
	if err == nil {
		t.Error("expected error for invalid TOML")
	}
}
