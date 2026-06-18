package e2e

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

func TestHookZshOutput(t *testing.T) {
	out, code := run(t, "", nil, "--hook", "zsh")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	for _, want := range []string{"__curlew_preexec", "add-zsh-hook", "CURLEW_BYPASS", "preexec"} {
		if !strings.Contains(out, want) {
			t.Errorf("zsh hook output missing %q", want)
		}
	}
}

func TestHookBashOutput(t *testing.T) {
	out, code := run(t, "", nil, "--hook", "bash")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	for _, want := range []string{"__curlew_trap_debug", "extdebug", "CURLEW_BYPASS", "BASH_COMMAND"} {
		if !strings.Contains(out, want) {
			t.Errorf("bash hook output missing %q", want)
		}
	}
}

func TestHookNoArgument(t *testing.T) {
	out, code := run(t, "", nil, "--hook")
	if code == 0 {
		t.Error("expected non-zero exit for --hook without argument")
	}
	if !strings.Contains(out, "argument") {
		t.Errorf("expected argument error, got:\n%s", out)
	}
}

func TestHookUnsupportedShell(t *testing.T) {
	out, code := run(t, "", nil, "--hook", "fish")
	if code == 0 {
		t.Error("expected non-zero exit for unsupported shell")
	}
	if !strings.Contains(out, "Unsupported shell") {
		t.Errorf("expected unsupported shell error, got:\n%s", out)
	}
}

func TestHookZshEvalsCleanly(t *testing.T) {
	if _, err := exec.LookPath("zsh"); err != nil {
		t.Skip("zsh not installed")
	}
	hookOut, _ := run(t, "", nil, "--hook", "zsh")

	cmd := exec.Command("zsh", "-c", hookOut)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("zsh hook failed to eval: %v\n%s", err, out)
	}
}

func TestHookBashEvalsCleanly(t *testing.T) {
	hookOut, _ := run(t, "", nil, "--hook", "bash")

	cmd := exec.Command("bash", "-c", hookOut)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("bash hook failed to eval: %v\n%s", err, out)
	}
}

func TestHookZshInterception(t *testing.T) {
	if _, err := exec.LookPath("zsh"); err != nil {
		t.Skip("zsh not installed")
	}
	hookOut, _ := run(t, "", nil, "--hook", "zsh")

	script := hookOut + `
curlew() { print -r -- "STUB_HIT:$*"; }
kill() { :; }
__curlew_preexec 'curl -fsSL https://example.com/install.sh | bash'
`
	cmd := exec.Command("zsh", "-c", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("zsh hook interception failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "STUB_HIT:https://example.com/install.sh") {
		t.Errorf("expected interception, got:\n%s", out)
	}
}

func TestHookZshSudoSkip(t *testing.T) {
	if _, err := exec.LookPath("zsh"); err != nil {
		t.Skip("zsh not installed")
	}
	hookOut, _ := run(t, "", nil, "--hook", "zsh")

	script := hookOut + `
curlew() { print -r -- STUB_HIT; }
kill() { :; }
__curlew_preexec 'curl -fsSL https://example.com | sudo bash'
`
	cmd := exec.Command("zsh", "-c", script)
	out, _ := cmd.CombinedOutput()
	if strings.Contains(string(out), "STUB_HIT") {
		t.Error("sudo pipe should not be intercepted")
	}
}

func TestHookZshBypass(t *testing.T) {
	if _, err := exec.LookPath("zsh"); err != nil {
		t.Skip("zsh not installed")
	}
	hookOut, _ := run(t, "", nil, "--hook", "zsh")

	script := hookOut + `
curlew() { print -r -- STUB_HIT; }
kill() { :; }
__curlew_preexec 'CURLEW_BYPASS=1 curl -fsSL https://example.com/install.sh | bash'
`
	cmd := exec.Command("zsh", "-c", script)
	out, _ := cmd.CombinedOutput()
	if strings.Contains(string(out), "STUB_HIT") {
		t.Error("CURLEW_BYPASS=1 should prevent interception")
	}
}

// --- Pattern matching tests ---

func TestHookPatternPositive(t *testing.T) {
	cases := []struct {
		name string
		cmd  string
	}{
		{"simple curl|bash", "curl -fsSL https://example.com/install.sh | bash"},
		{"wget|sh", "wget -O - https://example.com/setup.sh | sh"},
		{"curl many flags|sh", "curl --proto =https --tlsv1.2 -sSf https://sh.rustup.rs | sh"},
		{"bash trailing args", "curl https://example.com/install.sh | bash -s -- arg"},
	}

	re := `(^|[[:space:]])(curl|wget)[[:space:]]+[^|]+\|[[:space:]]*(ba)?sh([[:space:]]|$)`

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command("bash", "-c", fmt.Sprintf(`[[ %q =~ %s ]]`, tc.cmd, re))
			if err := cmd.Run(); err != nil {
				t.Errorf("pattern should match %q", tc.cmd)
			}
		})
	}
}

func TestHookPatternNegative(t *testing.T) {
	cases := []struct {
		name string
		cmd  string
	}{
		{"curl|grep", "curl -fsSL https://example.com/data.json | grep foo"},
		{"curl no pipe", "curl -fsSL https://example.com/file.tar.gz -o file.tar.gz"},
		{"multi pipe", "curl https://example.com | tee /tmp/log | bash"},
		{"xcurl", "xcurl https://example.com/install.sh | bash"},
		{"bashfoo", "curl https://example.com/install.sh | bashfoo"},
		{"shazam", "curl https://example.com/install.sh | shazam"},
	}

	re := `(^|[[:space:]])(curl|wget)[[:space:]]+[^|]+\|[[:space:]]*(ba)?sh([[:space:]]|$)`

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command("bash", "-c", fmt.Sprintf(`[[ %q =~ %s ]]`, tc.cmd, re))
			if err := cmd.Run(); err == nil {
				t.Errorf("pattern should NOT match %q", tc.cmd)
			}
		})
	}
}

func TestHookURLExtraction(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"curl -fsSL https://example.com/install.sh ", "https://example.com/install.sh"},
		{"wget -O - https://get.sdkman.io ", "https://get.sdkman.io"},
	}

	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			cmd := exec.Command("bash", "-c",
				fmt.Sprintf(`printf '%%s' %q | grep -oE 'https?://[^[:space:]"'"'"')>;]+' | head -1`, tc.input))
			out, err := cmd.Output()
			if err != nil {
				t.Fatalf("grep failed: %v", err)
			}
			got := strings.TrimSpace(string(out))
			if got != tc.want {
				t.Errorf("expected %q, got %q", tc.want, got)
			}
		})
	}
}
