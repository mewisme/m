package transaction_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/mewisme/mew/internal/fsx"
	"github.com/mewisme/mew/internal/jsonfile"
	"github.com/mewisme/mew/internal/transaction"
)

const preflightProcEnv = "MEW_TXN_PREFLIGHT_PROC"

func TestPreflightProcStaleLockAndIncomplete(t *testing.T) {
	if role := os.Getenv(preflightProcEnv); role != "" {
		runPreflightProcChild(t, role)
		return
	}
	if testing.Short() {
		t.Skip("preflight proc test")
	}
	root := t.TempDir()
	writeIncompleteTxn(t, root, "crash", transaction.StateStaging)
	lockDir := transaction.LockPath(root)
	if err := os.MkdirAll(lockDir, 0o755); err != nil {
		t.Fatal(err)
	}
	doc := transaction.LockDocument{
		SchemaVersion: 3,
		LockID:        "dead",
		PID:           999999,
		ProcessStart:  1,
		TxnID:         "crash",
		CreatedAt:     time.Now().UTC().Add(-time.Minute),
		ProjectRoot:   root,
	}
	raw, _ := jsonfile.Marshal(doc)
	if err := os.WriteFile(filepath.Join(lockDir, fsx.OwnerFileName), raw, 0o644); err != nil {
		t.Fatal(err)
	}

	exe, err := exec.LookPath(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe, "-test.run=^TestPreflightProcStaleLockAndIncomplete$", "-test.count=1")
	cmd.Env = append(os.Environ(),
		preflightProcEnv+"=begin",
		"MEW_PREFLIGHT_ROOT="+root,
	)
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	crashDir := filepath.Join(transaction.TxnRoot(root), "crash")
	if _, err := os.Stat(crashDir); !os.IsNotExist(err) {
		t.Fatal("stale crash txn dir should be removed after recovery")
	}

	txns, err := transaction.ScanIncompleteTxns(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(txns) != 1 {
		t.Fatalf("expected 1 incomplete txn from BeginMutation, got %d", len(txns))
	}
	if txns[0].ID == "crash" {
		t.Fatal("incomplete txn should be new BeginMutation id, not crash")
	}
	if txns[0].State != transaction.StateStaging {
		t.Fatalf("expected staging state, got %s", txns[0].State)
	}

	// Child exited without Finish; lock dir remains with stale owner.
	if _, err := os.Stat(lockDir); err != nil {
		t.Fatal("expected stale lock dir after child exit without Finish")
	}
}

func runPreflightProcChild(t *testing.T, role string) {
	t.Helper()
	root := os.Getenv("MEW_PREFLIGHT_ROOT")
	if root == "" {
		t.Fatal("missing MEW_PREFLIGHT_ROOT")
	}
	switch role {
	case "begin":
		ctx := context.Background()
		_, err := transaction.BeginMutation(ctx, root)
		if err != nil {
			t.Fatal(err)
		}
		// Exit without Finish so lock becomes stale and incomplete txn remains.
	default:
		t.Fatalf("unknown role %q", role)
	}
}
