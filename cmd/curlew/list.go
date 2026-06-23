package main

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/ketsugi/curlew/internal/config"
	"github.com/ketsugi/curlew/internal/ledger"
	"github.com/spf13/cobra"
)

func listCmd() *cobra.Command {
	var executedOnly bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List previously examined scripts",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(executedOnly)
		},
	}

	cmd.Flags().BoolVar(&executedOnly, "executed", false, "Show only scripts that were executed")
	return cmd
}

func runList(executedOnly bool) error {
	ledgerDir := config.LedgerDir()
	if ledgerDir == "" {
		return fmt.Errorf("cannot determine ledger path (no home directory)")
	}

	l, err := ledger.New(ledgerDir)
	if err != nil {
		return err
	}

	entries, err := l.List()
	if err != nil {
		return err
	}

	if executedOnly {
		filtered := entries[:0]
		for _, e := range entries {
			if e.Executed {
				filtered = append(filtered, e)
			}
		}
		entries = filtered
	}

	if len(entries) == 0 {
		fmt.Fprintln(os.Stderr, "No scripts recorded yet.")
		return nil
	}

	// Width for the right-aligned RUNS column: header vs the widest count.
	runsW := len("RUNS")
	for _, e := range entries {
		if d := len(strconv.Itoa(e.RunCount)); d > runsW {
			runsW = d
		}
	}

	// Format plainly through tabwriter, then bold the header line. Feeding ANSI
	// to tabwriter would throw off its column-width math, so the styling is
	// applied after the layout is computed.
	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "URL\tSHA256\t%*s\tLAST SEEN\n", runsW, "RUNS")
	for _, e := range entries {
		fmt.Fprintf(w, "%s\t%s\t%*d\t%s\n",
			e.URL,
			truncate(e.SHA256, 12),
			runsW, e.RunCount,
			e.LastRun.Format("2006-01-02 15:04"),
		)
	}
	w.Flush()

	header, rows, _ := strings.Cut(buf.String(), "\n")
	fmt.Printf("\033[1m%s\033[0m\n%s", header, rows)
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
