package transaction_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/transaction"
)

func writeIncompleteTxn(t *testing.T, root, id, state string) {
	t.Helper()
	txnRoot := transaction.TxnRoot(root)
	dir := filepath.Join(txnRoot, id)
	if err := os.MkdirAll(filepath.Join(dir, "stage"), 0o755); err != nil {
		t.Fatal(err)
	}
	doc := &transaction.Document{
		SchemaVersion: transaction.SchemaVersion,
		ID:            id,
		ProjectRoot:   root,
		State:         state,
	}
	data, err := transaction.Encode(doc)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "journal.000001.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScanIncompleteTxnsStates(t *testing.T) {
	root := t.TempDir()
	writeIncompleteTxn(t, root, "staging1", transaction.StateStaging)
	writeIncompleteTxn(t, root, "validated1", transaction.StateValidated)
	writeIncompleteTxn(t, root, "committing1", transaction.StateCommitting)
	writeIncompleteTxn(t, root, "done1", transaction.StateCommitted)
	writeIncompleteTxn(t, root, "abort1", transaction.StateAborted)

	got, err := transaction.ScanIncompleteTxns(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 incomplete, got %d", len(got))
	}
}

func TestScanIncompleteTxnsMissingCurrent(t *testing.T) {
	root := t.TempDir()
	writeIncompleteTxn(t, root, "orphan", transaction.StateStaging)
	if _, err := os.Stat(transaction.CurrentPath(root)); !os.IsNotExist(err) {
		t.Fatal("current should be missing")
	}
	got, err := transaction.ScanIncompleteTxns(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "orphan" {
		t.Fatalf("got %+v", got)
	}
}

func TestResolveAuthoritativeIncompleteMultipleCommitting(t *testing.T) {
	root := t.TempDir()
	writeIncompleteTxn(t, root, "c1", transaction.StateCommitting)
	writeIncompleteTxn(t, root, "c2", transaction.StateCommitting)
	txns, err := transaction.ScanIncompleteTxns(root)
	if err != nil {
		t.Fatal(err)
	}
	_, err = transaction.ResolveAuthoritativeIncomplete(txns)
	if err == nil {
		t.Fatal("expected integrity error")
	}
	if apperr.CodeOf(err) != apperr.Integrity {
		t.Fatalf("code=%s", apperr.CodeOf(err))
	}
}

func TestResolveAuthoritativeIncompletePriority(t *testing.T) {
	root := t.TempDir()
	writeIncompleteTxn(t, root, "staging", transaction.StateStaging)
	writeIncompleteTxn(t, root, "committing", transaction.StateCommitting)
	txns, err := transaction.ScanIncompleteTxns(root)
	if err != nil {
		t.Fatal(err)
	}
	auth, err := transaction.ResolveAuthoritativeIncomplete(txns)
	if err != nil {
		t.Fatal(err)
	}
	if auth == nil || auth.ID != "committing" {
		t.Fatalf("got %+v", auth)
	}
}

func TestBeginMutationRecoversStaging(t *testing.T) {
	root := t.TempDir()
	writeIncompleteTxn(t, root, "old", transaction.StateStaging)
	ctx := context.Background()
	run, err := transaction.BeginMutation(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	if run.ID == "old" {
		t.Fatal("expected new txn id")
	}
	txns, err := transaction.ScanIncompleteTxns(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(txns) != 1 || txns[0].ID != run.ID {
		t.Fatalf("incomplete after begin: %+v", txns)
	}
	_ = run.Finish(false)
}

func TestRecoverScannedIdempotent(t *testing.T) {
	root := t.TempDir()
	writeIncompleteTxn(t, root, "stale", transaction.StateValidated)
	ctx := context.Background()
	if err := transaction.RecoverScanned(ctx, root, transaction.RecoverScannedOpts{}); err != nil {
		t.Fatal(err)
	}
	if err := transaction.RecoverScanned(ctx, root, transaction.RecoverScannedOpts{}); err != nil {
		t.Fatal(err)
	}
	txns, err := transaction.ScanIncompleteTxns(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(txns) != 0 {
		t.Fatalf("expected clean state, got %d", len(txns))
	}
}
