package transaction_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/m/internal/transaction"
)

func TestClearCurrentVerifiedSuccess(t *testing.T) {
	root := t.TempDir()
	txnRoot := transaction.TxnRoot(root)
	if err := os.MkdirAll(txnRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(transaction.CurrentPath(root), []byte("abc\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(txnRoot, "current.000001"), []byte("abc\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(txnRoot, "current.head"), []byte(`{"generation":1,"checksum":"x"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := transaction.ClearCurrentVerifiedForTest(root)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Verified {
		t.Fatal("expected verified cleanup")
	}
	if _, err := os.Stat(transaction.CurrentPath(root)); !os.IsNotExist(err) {
		t.Fatal("legacy current should be removed")
	}
}

func TestClearCurrentVerifiedMalformedGen(t *testing.T) {
	root := t.TempDir()
	txnRoot := transaction.TxnRoot(root)
	if err := os.MkdirAll(txnRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(txnRoot, "current.foo"), []byte("junk\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := transaction.ClearCurrentVerifiedForTest(root)
	if err == nil {
		t.Fatal("expected malformed generation error")
	}
	if result.Verified {
		t.Fatal("should not verify with malformed generation file")
	}
}
