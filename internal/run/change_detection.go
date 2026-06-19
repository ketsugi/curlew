package run

import (
	"github.com/ketsugi/curlew/internal/config"
	"github.com/ketsugi/curlew/internal/ledger"
)

type changeStatus int

const (
	changeNew       changeStatus = iota // never seen this URL before
	changeUnchanged                     // same hash as last time
	changeModified                      // hash differs from last time
)

// checkForChanges compares the current script hash against the ledger.
// Returns changeNew for local files or unknown URLs.
func checkForChanges(target string, currentHash string) changeStatus {
	if !isURL(target) {
		return changeNew
	}

	ledgerDir := config.LedgerDir()
	if ledgerDir == "" {
		return changeNew
	}

	l, err := ledger.New(ledgerDir)
	if err != nil {
		return changeNew
	}

	entry, err := l.Lookup(target)
	if err != nil || entry == nil {
		return changeNew
	}

	if entry.SHA256 == currentHash {
		return changeUnchanged
	}
	return changeModified
}
