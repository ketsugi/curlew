package run

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/ketsugi/curlew/internal/staticanalysis"
)

// notableCategories trigger the expanded, per-finding breakdown. The rest get
// a compact one-line tally.
var notableCategories = map[staticanalysis.Category]bool{
	staticanalysis.PrivEsc:     true,
	staticanalysis.Persistence: true,
	staticanalysis.Dangerous:   true,
	staticanalysis.Obfuscation: true,
}

// reportStaticAnalysis runs structural analysis on the script and prints a
// summary. Output scales: a one-line tally for routine scripts, an expanded
// per-finding list when notable categories (privilege escalation, persistence,
// dangerous ops, obfuscation) are present. A parse failure (e.g. non-shell
// script) is silently skipped — the AI pass and manual inspection still apply.
func reportStaticAnalysis(script []byte) {
	report, err := staticanalysis.Analyze(script)
	if err != nil || len(report.Findings) == 0 {
		return
	}

	counts := map[staticanalysis.Category]int{}
	notable := false
	for _, f := range report.Findings {
		counts[f.Category]++
		if notableCategories[f.Category] {
			notable = true
		}
	}

	info("Static analysis: %s", summaryLine(counts))

	if !notable {
		return
	}

	// Expanded breakdown for notable categories.
	for _, cat := range report.Categories() {
		if !notableCategories[cat] {
			continue
		}
		for _, f := range report.Findings {
			if f.Category != cat {
				continue
			}
			fmt.Fprintf(os.Stderr, "    \033[1;33m%s\033[0m L%d: %s\n", cat, f.Line, f.Detail)
		}
	}
}

// summaryLine renders the per-category counts as a compact comma list, ordered
// by category for stable output.
func summaryLine(counts map[staticanalysis.Category]int) string {
	cats := make([]staticanalysis.Category, 0, len(counts))
	for c := range counts {
		cats = append(cats, c)
	}
	sort.Slice(cats, func(i, j int) bool { return cats[i] < cats[j] })

	parts := make([]string, 0, len(cats))
	for _, c := range cats {
		parts = append(parts, fmt.Sprintf("%d %s", counts[c], pluralize(c.String(), counts[c])))
	}
	return strings.Join(parts, ", ")
}

func pluralize(label string, n int) string {
	if n == 1 {
		return label
	}
	// Naive pluralization is fine for our fixed label set.
	return label + "s"
}
