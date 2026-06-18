package ai

import (
	"os"
	"strings"
	"testing"
)

func setEnv(t *testing.T, key, val string) {
	t.Helper()
	old, existed := os.LookupEnv(key)
	os.Setenv(key, val)
	t.Cleanup(func() {
		if existed {
			os.Setenv(key, old)
		} else {
			os.Unsetenv(key)
		}
	})
}

func clearEnv(t *testing.T, keys ...string) {
	t.Helper()
	for _, k := range keys {
		setEnv(t, k, "")
		os.Unsetenv(k)
	}
}

func TestResolveCommand_DefaultClaude(t *testing.T) {
	clearEnv(t, "CURLEW_AI", "CURLEW_MODEL", "CURLEW_AI_CMD", "CURLEW_CLAUDE_CMD")
	cmd, err := ResolveCommand()
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(cmd, " ")
	if got != "claude --model sonnet --print" {
		t.Errorf("expected 'claude --model sonnet --print', got %q", got)
	}
}

func TestResolveCommand_ClaudeWithModel(t *testing.T) {
	clearEnv(t, "CURLEW_AI_CMD", "CURLEW_CLAUDE_CMD")
	setEnv(t, "CURLEW_AI", "claude")
	setEnv(t, "CURLEW_MODEL", "opus")
	cmd, err := ResolveCommand()
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(cmd, " ")
	if got != "claude --model opus --print" {
		t.Errorf("expected 'claude --model opus --print', got %q", got)
	}
}

func TestResolveCommand_ClaudeWithCustomBinary(t *testing.T) {
	clearEnv(t, "CURLEW_AI_CMD")
	setEnv(t, "CURLEW_AI", "claude")
	setEnv(t, "CURLEW_MODEL", "")
	setEnv(t, "CURLEW_CLAUDE_CMD", "/opt/mock-claude")
	cmd, err := ResolveCommand()
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(cmd, " ")
	if got != "/opt/mock-claude --model sonnet --print" {
		t.Errorf("expected '/opt/mock-claude --model sonnet --print', got %q", got)
	}
}

func TestResolveCommand_Ollama(t *testing.T) {
	clearEnv(t, "CURLEW_AI_CMD", "CURLEW_CLAUDE_CMD")
	setEnv(t, "CURLEW_AI", "ollama")
	setEnv(t, "CURLEW_MODEL", "llama3")
	cmd, err := ResolveCommand()
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(cmd, " ")
	if got != "ollama run llama3" {
		t.Errorf("expected 'ollama run llama3', got %q", got)
	}
}

func TestResolveCommand_OllamaNoModel(t *testing.T) {
	clearEnv(t, "CURLEW_AI_CMD", "CURLEW_CLAUDE_CMD", "CURLEW_MODEL")
	setEnv(t, "CURLEW_AI", "ollama")
	_, err := ResolveCommand()
	if err == nil {
		t.Error("expected error for ollama without model")
	}
	if !strings.Contains(err.Error(), "CURLEW_MODEL") {
		t.Errorf("expected error to mention CURLEW_MODEL, got: %v", err)
	}
}

func TestResolveCommand_AICmdOverride(t *testing.T) {
	setEnv(t, "CURLEW_AI", "claude")
	setEnv(t, "CURLEW_MODEL", "opus")
	setEnv(t, "CURLEW_AI_CMD", "my-llm --chat")
	cmd, err := ResolveCommand()
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(cmd, " ")
	if got != "my-llm --chat" {
		t.Errorf("expected 'my-llm --chat', got %q", got)
	}
}

func TestResolveCommand_UnknownBackend(t *testing.T) {
	clearEnv(t, "CURLEW_AI_CMD", "CURLEW_CLAUDE_CMD", "CURLEW_MODEL")
	setEnv(t, "CURLEW_AI", "bogus")
	_, err := ResolveCommand()
	if err == nil {
		t.Error("expected error for unknown backend")
	}
	if !strings.Contains(err.Error(), "bogus") {
		t.Errorf("expected error to mention 'bogus', got: %v", err)
	}
}
