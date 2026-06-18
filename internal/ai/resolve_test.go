package ai

import (
	"strings"
	"testing"
)

func TestResolveCommand_DefaultClaude(t *testing.T) {
	cmd, err := ResolveCommand("", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(cmd, " ")
	if got != "claude --model sonnet --print" {
		t.Errorf("expected 'claude --model sonnet --print', got %q", got)
	}
}

func TestResolveCommand_ClaudeWithModel(t *testing.T) {
	cmd, err := ResolveCommand("claude", "opus", "", "")
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(cmd, " ")
	if got != "claude --model opus --print" {
		t.Errorf("expected 'claude --model opus --print', got %q", got)
	}
}

func TestResolveCommand_ClaudeWithCustomBinary(t *testing.T) {
	cmd, err := ResolveCommand("claude", "", "", "/opt/mock-claude")
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(cmd, " ")
	if got != "/opt/mock-claude --model sonnet --print" {
		t.Errorf("expected '/opt/mock-claude --model sonnet --print', got %q", got)
	}
}

func TestResolveCommand_Ollama(t *testing.T) {
	cmd, err := ResolveCommand("ollama", "llama3", "", "")
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(cmd, " ")
	if got != "ollama run llama3" {
		t.Errorf("expected 'ollama run llama3', got %q", got)
	}
}

func TestResolveCommand_OllamaNoModel(t *testing.T) {
	_, err := ResolveCommand("ollama", "", "", "")
	if err == nil {
		t.Error("expected error for ollama without model")
	}
}

func TestResolveCommand_AICmdOverride(t *testing.T) {
	cmd, err := ResolveCommand("claude", "opus", "my-llm --chat", "")
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(cmd, " ")
	if got != "my-llm --chat" {
		t.Errorf("expected 'my-llm --chat', got %q", got)
	}
}

func TestResolveCommand_UnknownBackend(t *testing.T) {
	_, err := ResolveCommand("bogus", "", "", "")
	if err == nil {
		t.Error("expected error for unknown backend")
	}
	if !strings.Contains(err.Error(), "bogus") {
		t.Errorf("expected error to mention 'bogus', got: %v", err)
	}
}
