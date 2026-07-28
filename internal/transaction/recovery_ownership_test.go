package transaction_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/transaction"
)

func TestRecoverScannedPreservesOuterLock(t *testing.T) {
	root := t.TempDir()
	txnRoot := transaction.TxnRoot(root)
	dir := filepath.Join(txnRoot, "stale")
	if err := os.MkdirAll(filepath.Join(dir, "stage"), 0o755); err != nil {
		t.Fatal(err)
	}
	doc := &transaction.Document{
		SchemaVersion: transaction.SchemaVersion,
		ID:            "stale",
		ProjectRoot:   root,
		State:         transaction.StateStaging,
	}
	data, err := transaction.Encode(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "journal.000001.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	if err := transaction.AcquireProjectLock(ctx, root, "outer"); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = transaction.ReleaseProjectLock(root, "outer") }()

	if err := transaction.RecoverScanned(ctx, root, transaction.RecoverScannedOpts{SkipTakeover: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(transaction.LockPath(root)); err != nil {
		t.Fatal("outer lock should remain after recovery under SkipTakeover")
	}
}

func TestReleaseProjectLockNotOwnerReturnsError(t *testing.T) {
	root := t.TempDir()
	ctx := context.Background()
	if err := transaction.AcquireProjectLock(ctx, root, "owner"); err != nil {
		t.Fatal(err)
	}
	if err := transaction.ReleaseProjectLock(root, "other"); err == nil {
		t.Fatal("expected not-owner error")
	}
	if err := transaction.ReleaseProjectLock(root, "owner"); err != nil {
		t.Fatal(err)
	}
}
