package e2e

import (
	"fmt"
	"os/exec"
	"regexp"
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

func TestHookZshInterceptsCommandSubstitution(t *testing.T) {
	if _, err := exec.LookPath("zsh"); err != nil {
		t.Skip("zsh not installed")
	}
	hookOut, _ := run(t, "", nil, "--hook", "zsh")

	// The Homebrew installer form: bash -c "$(curl ...)"
	script := hookOut + `
curlew() { print -r -- "STUB_HIT:$*"; }
kill() { :; }
__curlew_preexec '/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"'
`
	cmd := exec.Command("zsh", "-c", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("zsh hook failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "STUB_HIT:https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh") {
		t.Errorf("expected interception of command-substitution form, got:\n%s", out)
	}
}

func TestHookBashWarnsCommandSubstitution(t *testing.T) {
	hookOut, _ := run(t, "", nil, "--hook", "bash")

	// Drive the REAL trap function (from the emitted hook) by setting
	// BASH_COMMAND, which is what the trap reads. The warn-only path must emit
	// a pastable `curlew <url>` and must NOT invoke the real curlew, since bash
	// can't block this form.
	script := hookOut + `
curlew() { echo "SHOULD_NOT_RUN"; }
BASH_COMMAND='/bin/bash -c "$(curl -fsSL https://example.com/install.sh)"' __curlew_trap_debug
`
	cmd := exec.Command("bash", "-c", script)
	out, _ := cmd.CombinedOutput()
	if !strings.Contains(string(out), "curlew https://example.com/install.sh") {
		t.Errorf("expected pastable curlew command in warning, got:\n%s", out)
	}
	if !strings.Contains(string(out), "cannot intercept") {
		t.Errorf("expected the warn-only explanation, got:\n%s", out)
	}
	if strings.Contains(string(out), "SHOULD_NOT_RUN") {
		t.Error("bash warn-only path must not invoke curlew")
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

func TestHookSubstPattern(t *testing.T) {
	// The hook's bash ERE is POSIX-ECMA-compatible for this pattern; Go's
	// regexp evaluates it identically. Matching in-process avoids handing a
	// command containing $(curl ...) to a shell, which would expand and run it.
	substRe := regexp.MustCompile(`(^|[[:space:]])(/[^[:space:]]*)?(ba)?sh[[:space:]]+-c[[:space:]].*\$\((curl|wget)[[:space:]]`)

	positives := []struct{ name, cmd string }{
		{"homebrew", `/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"`},
		{"plain bash -c curl", `bash -c "$(curl -fsSL https://example.com/i.sh)"`},
		{"sh -c wget", `sh -c "$(wget -O- https://example.com/i.sh)"`},
		{"usr/bin/sh", `/usr/bin/sh -c "$(curl https://x.com)"`},
	}
	for _, tc := range positives {
		t.Run("match/"+tc.name, func(t *testing.T) {
			if !substRe.MatchString(tc.cmd) {
				t.Errorf("subst pattern should match %q", tc.cmd)
			}
		})
	}

	negatives := []struct{ name, cmd string }{
		{"bash -c echo", `bash -c "echo hello"`},
		{"no -c", `bash "$(curl https://x.com)"`},
		{"git commit -c", `git commit -c HEAD`},
		{"comment mention", `echo "run bash -c with curl"`},
	}
	for _, tc := range negatives {
		t.Run("nomatch/"+tc.name, func(t *testing.T) {
			if substRe.MatchString(tc.cmd) {
				t.Errorf("subst pattern should NOT match %q", tc.cmd)
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
