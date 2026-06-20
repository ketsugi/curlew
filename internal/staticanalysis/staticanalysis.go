// Package staticanalysis performs structural analysis of shell scripts by
// walking the AST produced by mvdan.cc/sh. It surfaces what a script would do
// — network calls, file writes, package installs, privilege escalation,
// persistence, dangerous operations, and obfuscation — without executing it.
//
// This is the deterministic, dependency-light layer beneath the AI analysis:
// it reports literal facts about the script's structure. It cannot resolve
// dynamic behavior (eval/base64 payloads, full variable dataflow); those are
// flagged as suspicious but not decoded.
package staticanalysis

import (
	"regexp"
	"strings"

	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/syntax"
)

// Category classifies a static-analysis finding.
type Category int

const (
	Network Category = iota
	URL
	FileWrite
	PackageInstall
	PrivEsc
	Persistence
	Dangerous
	Obfuscation
)

// String returns a human-readable category label.
func (c Category) String() string {
	switch c {
	case Network:
		return "Network call"
	case URL:
		return "URL"
	case FileWrite:
		return "File write"
	case PackageInstall:
		return "Package install"
	case PrivEsc:
		return "Privilege escalation"
	case Persistence:
		return "Persistence"
	case Dangerous:
		return "Dangerous operation"
	case Obfuscation:
		return "Obfuscation"
	default:
		return "Unknown"
	}
}

// Finding is a single structural observation about the script.
type Finding struct {
	Category Category
	Line     uint
	Detail   string // the command text, URL, or path that triggered it
}

// Report is the result of analyzing a script.
type Report struct {
	Findings []Finding
}

// Categories returns the distinct categories present, in declaration order.
func (r *Report) Categories() []Category {
	seen := map[Category]bool{}
	var out []Category
	for c := Network; c <= Obfuscation; c++ {
		for _, f := range r.Findings {
			if f.Category == c && !seen[c] {
				seen[c] = true
				out = append(out, c)
			}
		}
	}
	return out
}

var (
	urlRe        = regexp.MustCompile(`https?://[^\s"'` + "`" + `)>;|]+`)
	netCmds      = map[string]bool{"curl": true, "wget": true, "fetch": true}
	fileWriteCmd = map[string]bool{"cp": true, "mv": true, "mkdir": true, "install": true, "touch": true, "tee": true, "dd": true, "ln": true}
	privEscCmd   = map[string]bool{"sudo": true, "doas": true, "chmod": true, "chown": true, "su": true}
	obfusCmd     = map[string]bool{"eval": true, "base64": true, "xxd": true}
	// Persistence-relevant shell profile / scheduler targets.
	profileRe  = regexp.MustCompile(`\.(bashrc|bash_profile|zshrc|zprofile|profile|zshenv)$`)
	persistCmd = map[string]bool{"crontab": true, "systemctl": true, "launchctl": true, "schtasks": true}
	// Package managers whose "install"/"add" subcommand pulls software.
	pkgMgrs = map[string]bool{
		"apt": true, "apt-get": true, "yum": true, "dnf": true, "brew": true,
		"pip": true, "pip3": true, "npm": true, "yarn": true, "gem": true,
		"cargo": true, "go": true, "pacman": true, "apk": true, "snap": true,
	}
	pkgInstallVerb = map[string]bool{"install": true, "add": true, "-S": true}
)

// Analyze parses src as a shell script and returns its structural findings.
// Returns an error only if the script fails to parse.
func Analyze(src []byte) (*Report, error) {
	parser := syntax.NewParser()
	file, err := parser.Parse(strings.NewReader(string(src)), "")
	if err != nil {
		return nil, err
	}

	r := &Report{}
	syntax.Walk(file, func(node syntax.Node) bool {
		switch n := node.(type) {
		case *syntax.CallExpr:
			r.analyzeCall(n)
		case *syntax.Redirect:
			r.analyzeRedirect(n)
		case *syntax.Assign:
			r.analyzeAssign(n)
		}
		return true
	})
	return r, nil
}

func (r *Report) add(c Category, line uint, detail string) {
	r.Findings = append(r.Findings, Finding{Category: c, Line: line, Detail: detail})
}

// litString flattens a word to its literal text, or "" if it isn't a plain
// literal (e.g. contains expansions). Good enough for command-name and
// subcommand matching.
func litString(w *syntax.Word) string {
	if w == nil {
		return ""
	}
	s, err := expand.Literal(nil, w)
	if err != nil {
		// Fall back to concatenating literal parts (handles words with
		// expansions by ignoring the dynamic bits).
		var b strings.Builder
		for _, part := range w.Parts {
			if lit, ok := part.(*syntax.Lit); ok {
				b.WriteString(lit.Value)
			}
		}
		return b.String()
	}
	return s
}

func (r *Report) analyzeCall(call *syntax.CallExpr) {
	if len(call.Args) == 0 {
		return
	}
	line := call.Pos().Line()
	cmd := litString(call.Args[0])
	base := cmd
	if idx := strings.LastIndex(base, "/"); idx >= 0 {
		base = base[idx+1:]
	}

	// Collect arg literals once.
	args := make([]string, 0, len(call.Args)-1)
	for _, a := range call.Args[1:] {
		args = append(args, litString(a))
	}

	switch {
	case netCmds[base]:
		r.add(Network, line, commandText(call))
	case privEscCmd[base]:
		r.add(PrivEsc, line, commandText(call))
	case persistCmd[base]:
		r.add(Persistence, line, commandText(call))
	case obfusCmd[base]:
		r.add(Obfuscation, line, commandText(call))
	case fileWriteCmd[base]:
		r.add(FileWrite, line, commandText(call))
	}

	// "eval" is a keyword-ish builtin but still appears as a call arg[0].
	if base == "rm" && hasRecursiveForce(args) {
		r.add(Dangerous, line, commandText(call))
	}
	if base == "dd" || base == "mkfs" || strings.HasPrefix(base, "mkfs.") {
		r.add(Dangerous, line, commandText(call))
	}

	// Package install: <pkgmgr> <verb> ...
	if pkgMgrs[base] && len(args) > 0 {
		for _, a := range args {
			if pkgInstallVerb[a] {
				r.add(PackageInstall, line, commandText(call))
				break
			}
			// stop scanning at the first non-flag token that isn't a verb
			if !strings.HasPrefix(a, "-") {
				break
			}
		}
	}

	// URLs anywhere in the args.
	for _, a := range args {
		for _, u := range urlRe.FindAllString(a, -1) {
			r.add(URL, line, u)
		}
	}
}

func (r *Report) analyzeRedirect(rd *syntax.Redirect) {
	// Only output redirects that write to a file count as file writes.
	switch rd.Op {
	case syntax.RdrOut, syntax.AppOut, syntax.RdrAll, syntax.AppAll:
	default:
		return
	}
	target := litString(rd.Word)
	if target == "" {
		return // dynamic target — can't classify
	}
	// Skip /dev/null and other /dev/ sinks; those aren't meaningful writes.
	if target == "/dev/null" || strings.HasPrefix(target, "/dev/") {
		return
	}
	line := rd.Pos().Line()
	r.add(FileWrite, line, target)
	if profileRe.MatchString(target) {
		r.add(Persistence, line, target)
	}
}

func (r *Report) analyzeAssign(a *syntax.Assign) {
	if a.Value == nil {
		return
	}
	val := litString(a.Value)
	line := a.Pos().Line()
	for _, u := range urlRe.FindAllString(val, -1) {
		r.add(URL, line, u)
	}
}

func hasRecursiveForce(args []string) bool {
	recursive, force := false, false
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			continue
		}
		if a == "--recursive" {
			recursive = true
		}
		if a == "--force" {
			force = true
		}
		// bundled short flags: -rf, -fr, -Rf, etc.
		if !strings.HasPrefix(a, "--") {
			for _, c := range a[1:] {
				switch c {
				case 'r', 'R':
					recursive = true
				case 'f':
					force = true
				}
			}
		}
	}
	return recursive && force
}

// commandText renders a call expression back to a compact single-line string
// for display in findings.
func commandText(call *syntax.CallExpr) string {
	parts := make([]string, 0, len(call.Args))
	for _, a := range call.Args {
		s := litString(a)
		if s == "" {
			s = "…" // a dynamic word we couldn't flatten
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, " ")
}
