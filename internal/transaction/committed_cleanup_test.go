package transaction_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/m/internal/transaction"
)

func writeCommittedTxnWithCurrent(t *testing.T, root, id string) {
	t.Helper()
	txnRoot := transaction.TxnRoot(root)
	dir := filepath.Join(txnRoot, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	doc := &transaction.Document{
		SchemaVersion: transaction.SchemaVersion,
		ID:            id,
		ProjectRoot:   root,
		State:         transaction.StateCommitted,
	}
	data, err := transaction.Encode(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "journal.000001.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(transaction.CurrentPath(root), []byte(id+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRecoverCommittedCleanupClearsCurrent(t *testing.T) {
	root := t.TempDir()
	id := "committed-stale"
	writeCommittedTxnWithCurrent(t, root, id)
	ctx := context.Background()
	cleaned, err := transaction.RecoverCommittedCleanup(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	if cleaned != 1 {
		t.Fatalf("cleaned=%d", cleaned)
	}
	stale, err := transaction.ScanCommittedStale(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) != 0 {
		t.Fatalf("stale remaining: %+v", stale)
	}
}
