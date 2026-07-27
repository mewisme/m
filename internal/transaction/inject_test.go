package transaction_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/m/internal/transaction"
)

func TestInjectCommitMidPlanRollback(t *testing.T) {
	root := t.TempDir()
	liveLock := filepath.Join(root, "m.lock")
	if err := os.WriteFile(liveLock, []byte("old"), 0o644); err != nil {
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
	if err := os.WriteFile(filepath.Join(stage, "m.lock"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(stageNM, filepath.Join(stage, "node_modules")); err != nil {
		t.Fatal(err)
	}
	plan := []transaction.Op{
		{Kind: transaction.OpRename, Path: "m.lock", Backup: "stage/m.lock"},
		{Kind: transaction.OpRename, Path: "node_modules", Backup: "stage/node_modules"},
	}
	if err := txn.SetPlan(plan); err != nil {
		t.Fatal(err)
	}
	if err := txn.RecordBackup("m.lock"); err != nil {
		t.Fatal(err)
	}
	if err := txn.RecordBackup("node_modules"); err != nil {
		t.Fatal(err)
	}

	failAt := 1
	transaction.SetTestHook(func(phase string, opIndex int) error {
		if phase == "commit" && opIndex == failAt {
			return os.ErrPermission
		}
		return nil
	})
	t.Cleanup(func() { transaction.SetTestHook(nil) })

	if err := txn.Commit(ctx, nil); err == nil {
		t.Fatal("expected commit failure")
	}
	data, err := os.ReadFile(liveLock)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "old" {
		t.Fatalf("lock not restored: %q", data)
	}
}

func TestInjectRecoveryIdempotent(t *testing.T) {
	root := t.TempDir()
	liveLock := filepath.Join(root, "m.lock")
	if err := os.WriteFile(liveLock, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	txn := transaction.NewRunner(root)
	ctx := context.Background()
	if err := txn.Begin(ctx); err != nil {
		t.Fatal(err)
	}
	if err := txn.RecordBackup("m.lock"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(liveLock, []byte("partial"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := txn.SetState(transaction.StateCommitting); err != nil {
		t.Fatal(err)
	}
	if err := transaction.RecoverIncomplete(ctx, root); err != nil {
		t.Fatal(err)
	}
	if err := transaction.RecoverIncomplete(ctx, root); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(liveLock)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "old" {
		t.Fatalf("lock not restored after recovery: %q", data)
	}
}

func TestPartialNodeModulesRepair(t *testing.T) {
	root := t.TempDir()
	live := filepath.Join(root, "node_modules")
	backup := live + ".mew-old"
	if err := os.MkdirAll(filepath.Join(backup, "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(live); !os.IsNotExist(err) {
		t.Fatalf("live should be missing, got %v", err)
	}
	txn := transaction.NewRunner(root)
	ctx := context.Background()
	if err := txn.Begin(ctx); err != nil {
		t.Fatal(err)
	}
	if err := txn.SetState(transaction.StateCommitting); err != nil {
		t.Fatal(err)
	}
	if err := txn.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(live, "pkg")); err != nil {
		t.Fatalf("node_modules not repaired: %v", err)
	}
}

func TestProjectLockStale(t *testing.T) {
	root := t.TempDir()
	ctx := context.Background()
	if err := transaction.AcquireProjectLock(ctx, root, "a"); err != nil {
		t.Fatal(err)
	}
	if err := transaction.ReleaseProjectLock(root, "a"); err != nil {
		t.Fatal(err)
	}
	if err := transaction.AcquireProjectLock(ctx, root, "b"); err != nil {
		t.Fatal(err)
	}
	_ = transaction.ReleaseProjectLock(root, "b")
}

func TestInjectRollbackWithoutStagedArtifact(t *testing.T) {
	root := t.TempDir()
	liveLock := filepath.Join(root, "m.lock")
	if err := os.WriteFile(liveLock, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	txn := transaction.NewRunner(root)
	ctx := context.Background()
	if err := txn.Begin(ctx); err != nil {
		t.Fatal(err)
	}
	stage := txn.StagePath()
	if err := os.WriteFile(filepath.Join(stage, "m.lock"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan := []transaction.Op{{Kind: transaction.OpRename, Path: "m.lock", Backup: "stage/m.lock"}}
	if err := txn.SetPlan(plan); err != nil {
		t.Fatal(err)
	}
	if err := txn.RecordBackup("m.lock"); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(stage, "m.lock")); err != nil {
		t.Fatal(err)
	}
	if err := txn.SetState(transaction.StateCommitting); err != nil {
		t.Fatal(err)
	}
	if err := txn.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(liveLock)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "old" {
		t.Fatalf("backup-authoritative restore failed: %q", data)
	}
}

func TestInjectCommittedMarkerOnlyAfterPlan(t *testing.T) {
	root := t.TempDir()
	liveLock := filepath.Join(root, "m.lock")
	if err := os.WriteFile(liveLock, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	txn := transaction.NewRunner(root)
	ctx := context.Background()
	if err := txn.Begin(ctx); err != nil {
		t.Fatal(err)
	}
	stage := txn.StagePath()
	if err := os.WriteFile(filepath.Join(stage, "m.lock"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := txn.SetPlan([]transaction.Op{{Kind: transaction.OpRename, Path: "m.lock", Backup: "stage/m.lock"}}); err != nil {
		t.Fatal(err)
	}
	if err := txn.RecordBackup("m.lock"); err != nil {
		t.Fatal(err)
	}
	transaction.SetTestHook(func(phase string, opIndex int) error {
		if phase == "pre_committed" {
			doc := txn.Document()
			if doc.State == transaction.StateCommitted {
				return os.ErrInvalid
			}
		}
		return nil
	})
	t.Cleanup(func() { transaction.SetTestHook(nil) })
	if err := txn.Commit(ctx, nil); err != nil {
		t.Fatal(err)
	}
	doc := txn.Document()
	if doc.State != transaction.StateCommitted {
		t.Fatalf("state=%s", doc.State)
	}
}
