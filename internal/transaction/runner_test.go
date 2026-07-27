package transaction_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/m/internal/transaction"
)

func TestRunnerCommitRollbackPreservesLive(t *testing.T) {
	root := t.TempDir()
	live := filepath.Join(root, "m.lock")
	if err := os.WriteFile(live, []byte("old-lock"), 0o644); err != nil {
		t.Fatal(err)
	}
	stageNM := filepath.Join(root, "stage-nm")
	if err := os.MkdirAll(filepath.Join(stageNM, "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}

	txn := transaction.NewRunner(root)
	ctx := context.Background()
	if err := txn.Begin(ctx); err != nil {
		t.Fatal(err)
	}
	stage := txn.StagePath()
	if err := os.WriteFile(filepath.Join(stage, "m.lock"), []byte("new-lock"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(stageNM, filepath.Join(stage, "node_modules")); err != nil {
		t.Fatal(err)
	}
	if err := txn.RecordBackup("m.lock"); err != nil {
		t.Fatal(err)
	}
	transaction.SetTestHook(func(phase string, opIndex int) error {
		if phase == "commit" && opIndex == 1 {
			return os.ErrPermission
		}
		return nil
	})
	t.Cleanup(func() { transaction.SetTestHook(nil) })

	ops := []transaction.Op{
		{Kind: transaction.OpRename, Path: "m.lock", Backup: "stage/m.lock"},
	}
	err := txn.Commit(ctx, ops)
	if err == nil {
		t.Fatal("expected commit failure")
	}
	data, err := os.ReadFile(live)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "old-lock" {
		t.Fatalf("live lock changed: %q", data)
	}
}

func TestRunnerBackupAndRestore(t *testing.T) {
	root := t.TempDir()
	live := filepath.Join(root, "package.json")
	if err := os.WriteFile(live, []byte(`{"name":"a"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	txn := transaction.NewRunner(root)
	ctx := context.Background()
	if err := txn.Begin(ctx); err != nil {
		t.Fatal(err)
	}
	if err := txn.RecordBackup("package.json"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(live, []byte(`{"name":"b"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := txn.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(live)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"a"`) {
		t.Fatalf("restore failed: %s", data)
	}
}
