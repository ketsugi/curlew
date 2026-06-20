package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- DetectShell ---

func TestDetectShell_Zsh(t *testing.T) {
	if got := DetectShell("/bin/zsh"); got != "zsh" {
		t.Errorf("expected zsh, got %q", got)
	}
}

func TestDetectShell_Bash(t *testing.T) {
	if got := DetectShell("/usr/bin/bash"); got != "bash" {
		t.Errorf("expected bash, got %q", got)
	}
}

func TestDetectShell_ZshWithVersion(t *testing.T) {
	if got := DetectShell("/opt/homebrew/bin/zsh-5.9"); got != "zsh" {
		t.Errorf("expected zsh, got %q", got)
	}
}

func TestDetectShell_Unsupported(t *testing.T) {
	if got := DetectShell("/usr/bin/fish"); got != "" {
		t.Errorf("expected empty for fish, got %q", got)
	}
}

func TestDetectShell_Empty(t *testing.T) {
	if got := DetectShell(""); got != "" {
		t.Errorf("expected empty for empty input, got %q", got)
	}
}

// --- shell table consistency ---

// TestShellTable_Consistent guards the table-driven design: every shell entry
// must be detectable by its own binPrefix and have a resolvable rc path. Adding
// a half-wired shell entry fails here.
func TestShellTable_Consistent(t *testing.T) {
	t.Setenv("HOME", "/home/test")
	t.Setenv("ZDOTDIR", "")
	for _, s := range shells {
		if got := DetectShell("/usr/bin/" + s.binPrefix); got != s.name {
			t.Errorf("DetectShell(%q) = %q, want %q", s.binPrefix, got, s.name)
		}
		rc, err := RCFile(s.name)
		if err != nil {
			t.Errorf("RCFile(%q) errored: %v", s.name, err)
		}
		if rc == "" {
			t.Errorf("RCFile(%q) returned empty path", s.name)
		}
	}
}

// --- HookLine ---

func TestHookLine_Zsh(t *testing.T) {
	got := HookLine("zsh")
	if !strings.Contains(got, "curlew --hook zsh") {
		t.Errorf("expected zsh hook line, got %q", got)
	}
	if !strings.Contains(got, "eval") {
		t.Errorf("expected eval in hook line, got %q", got)
	}
}

func TestHookLine_Bash(t *testing.T) {
	got := HookLine("bash")
	if !strings.Contains(got, "curlew --hook bash") {
		t.Errorf("expected bash hook line, got %q", got)
	}
}

// --- RCFile ---

func TestRCFile_Zsh(t *testing.T) {
	t.Setenv("HOME", "/home/test")
	t.Setenv("ZDOTDIR", "")
	got, err := RCFile("zsh")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/home/test/.zshrc" {
		t.Errorf("expected /home/test/.zshrc, got %q", got)
	}
}

func TestRCFile_ZshWithZDOTDIR(t *testing.T) {
	t.Setenv("HOME", "/home/test")
	t.Setenv("ZDOTDIR", "/custom/zdotdir")
	got, err := RCFile("zsh")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/custom/zdotdir/.zshrc" {
		t.Errorf("expected /custom/zdotdir/.zshrc, got %q", got)
	}
}

func TestRCFile_Bash(t *testing.T) {
	t.Setenv("HOME", "/home/test")
	got, err := RCFile("bash")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/home/test/.bashrc" {
		t.Errorf("expected /home/test/.bashrc, got %q", got)
	}
}

func TestRCFile_Unsupported(t *testing.T) {
	_, err := RCFile("fish")
	if err == nil {
		t.Error("expected error for unsupported shell")
	}
}

// --- HookInstalled ---

func TestHookInstalled_NotPresent(t *testing.T) {
	rc := filepath.Join(t.TempDir(), ".zshrc")
	os.WriteFile(rc, []byte("# my zshrc\nexport PATH=/usr/bin\n"), 0o644)

	installed, err := HookInstalled(rc, "zsh")
	if err != nil {
		t.Fatal(err)
	}
	if installed {
		t.Error("expected hook not installed")
	}
}

func TestHookInstalled_Present(t *testing.T) {
	rc := filepath.Join(t.TempDir(), ".zshrc")
	os.WriteFile(rc, []byte("# my zshrc\neval \"$(curlew --hook zsh)\"\n"), 0o644)

	installed, err := HookInstalled(rc, "zsh")
	if err != nil {
		t.Fatal(err)
	}
	if !installed {
		t.Error("expected hook installed")
	}
}

func TestHookInstalled_MissingFile(t *testing.T) {
	rc := filepath.Join(t.TempDir(), ".zshrc")
	// file does not exist
	installed, err := HookInstalled(rc, "zsh")
	if err != nil {
		t.Fatal(err)
	}
	if installed {
		t.Error("expected hook not installed for missing file")
	}
}

// --- InstallHook ---

func TestInstallHook_AppendsToExisting(t *testing.T) {
	rc := filepath.Join(t.TempDir(), ".zshrc")
	os.WriteFile(rc, []byte("# my zshrc\nexport PATH=/usr/bin\n"), 0o644)

	if err := InstallHook(rc, "zsh"); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(rc)
	content := string(data)
	if !strings.Contains(content, "export PATH=/usr/bin") {
		t.Error("existing content should be preserved")
	}
	if !strings.Contains(content, "curlew --hook zsh") {
		t.Error("hook line should be appended")
	}
}

func TestInstallHook_CreatesFileIfMissing(t *testing.T) {
	rc := filepath.Join(t.TempDir(), ".bashrc")

	if err := InstallHook(rc, "bash"); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(rc)
	if err != nil {
		t.Fatalf("file should have been created: %v", err)
	}
	if !strings.Contains(string(data), "curlew --hook bash") {
		t.Error("hook line should be present")
	}
}

func TestInstallHook_CreatesParentDir(t *testing.T) {
	// rc file in a not-yet-existing directory (e.g. a custom $ZDOTDIR)
	rc := filepath.Join(t.TempDir(), "nested", "zdotdir", ".zshrc")

	if err := InstallHook(rc, "zsh"); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(rc)
	if err != nil {
		t.Fatalf("file should have been created: %v", err)
	}
	if !strings.Contains(string(data), "curlew --hook zsh") {
		t.Error("hook line should be present")
	}
}

func TestInstallHook_Idempotent(t *testing.T) {
	rc := filepath.Join(t.TempDir(), ".zshrc")
	os.WriteFile(rc, []byte("# my zshrc\n"), 0o644)

	if err := InstallHook(rc, "zsh"); err != nil {
		t.Fatal(err)
	}
	if err := InstallHook(rc, "zsh"); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(rc)
	count := strings.Count(string(data), "curlew --hook zsh")
	if count != 1 {
		t.Errorf("expected hook line exactly once, got %d", count)
	}
}

func TestInstallHook_PreservesTrailingNewline(t *testing.T) {
	rc := filepath.Join(t.TempDir(), ".zshrc")
	// File without trailing newline
	os.WriteFile(rc, []byte("export FOO=bar"), 0o644)

	if err := InstallHook(rc, "zsh"); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(rc)
	content := string(data)
	// The original line and the hook line should be on separate lines
	if strings.Contains(content, "export FOO=bareval") {
		t.Errorf("hook line ran into existing content: %q", content)
	}
}
