package main

import (
	"fmt"
	"os"
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

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "URL\tSHA256\tRUNS\tLAST SEEN")
	for _, e := range entries {
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\n",
			e.URL,
			truncate(e.SHA256, 12),
			e.RunCount,
			e.LastRun.Format("2006-01-02 15:04"),
		)
	}
	w.Flush()
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
