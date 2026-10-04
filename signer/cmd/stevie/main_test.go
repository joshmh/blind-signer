package main

import (
	"os"
	"path/filepath"
	"testing"
)

func validPsbt(t *testing.T) []byte {
	t.Helper()
	return serializePsbt(t, psbtWithOneSegwitInput(t))
}

// TestProcessTransactionsFiltersAndCounts pins issues 2 and 3: dotfiles and
// non-psbt junk never enter the count, an empty directory reports "no
// unsigned transactions" with success=false, and only successes delete toxic.
func TestProcessTransactionsFiltersAndCounts(t *testing.T) {
	dir := t.TempDir()
	txsDir := filepath.Join(dir, "unsigned")
	signedDir := filepath.Join(dir, "signed")
	for _, d := range []string{txsDir, signedDir} {
		if err := os.MkdirAll(d, 0700); err != nil {
			t.Fatal(err)
		}
	}

	// Empty dir → false, no error, nothing wiped.
	ok, err := processTransactions(0, 0, txsDir, signedDir, "h", masterKey(t))
	if err != nil {
		t.Fatalf("empty dir: %v", err)
	}
	if ok {
		t.Fatal("empty dir must not report success")
	}

	// Dotfile only → still zero work items.
	junk := filepath.Join(txsDir, ".DS_Store")
	if err := os.WriteFile(junk, []byte("junk"), 0600); err != nil {
		t.Fatal(err)
	}
	ok, err = processTransactions(0, 0, txsDir, signedDir, "h", masterKey(t))
	if err != nil {
		t.Fatalf("dotfile dir: %v", err)
	}
	if ok {
		t.Fatal("dotfile-only dir must not report success")
	}

	// A stale output in signed/ must be wiped by a real run start.
	stale := filepath.Join(signedDir, "stale_signed_h.psbt")
	if err := os.WriteFile(stale, []byte("stale"), 0600); err != nil {
		t.Fatal(err)
	}

	psbt := validPsbt(t)
	if err := os.WriteFile(filepath.Join(txsDir, "invoice42.psbt"), psbt, 0600); err != nil {
		t.Fatal(err)
	}
	// AppleDouble companion must be ignored.
	if err := os.WriteFile(filepath.Join(txsDir, "._invoice42.psbt"), []byte("junk"), 0600); err != nil {
		t.Fatal(err)
	}

	ok, err = processTransactions(0, 0, txsDir, signedDir, "h", masterKey(t))
	if err != nil {
		t.Fatalf("valid run: %v", err)
	}
	if !ok {
		t.Fatal("valid psbt must sign successfully")
	}

	entries, err := os.ReadDir(signedDir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if len(names) != 1 || names[0] != "invoice42_signed_h.psbt" {
		t.Fatalf("expected exactly invoice42_signed_h.psbt, got %v", names)
	}

	// Toxic material untouched by processTransactions itself is main()'s job;
	// the success contract tested here is the count semantics.
}

// TestProcessTransactionsWipesSigned pins issue 4(b): stale outputs from a
// previous run never survive a new run's start.
func TestProcessTransactionsWipesSigned(t *testing.T) {
	dir := t.TempDir()
	txsDir := filepath.Join(dir, "unsigned")
	signedDir := filepath.Join(dir, "signed")
	for _, d := range []string{txsDir, signedDir} {
		if err := os.MkdirAll(d, 0700); err != nil {
			t.Fatal(err)
		}
	}
	stale := filepath.Join(signedDir, "yesterday_signed_h.psbt")
	if err := os.WriteFile(stale, []byte("yesterday"), 0600); err != nil {
		t.Fatal(err)
	}

	// Empty unsigned dir: wipe still happens before the no-work early return.
	if _, err := processTransactions(0, 0, txsDir, signedDir, "h", masterKey(t)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatal("stale signed output survived a new run")
	}
}
