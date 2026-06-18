package hook

import (
	"strings"
	"testing"
)

func TestZshHook(t *testing.T) {
	h := ZshHook()

	checks := []string{
		"__curlew_preexec",
		"add-zsh-hook",
		"CURLEW_BYPASS",
		"preexec",
	}
	for _, want := range checks {
		if !strings.Contains(h, want) {
			t.Errorf("ZshHook() missing %q", want)
		}
	}
}

func TestBashHook(t *testing.T) {
	h := BashHook()

	checks := []string{
		"__curlew_trap_debug",
		"extdebug",
		"CURLEW_BYPASS",
		"BASH_COMMAND",
	}
	for _, want := range checks {
		if !strings.Contains(h, want) {
			t.Errorf("BashHook() missing %q", want)
		}
	}
}
