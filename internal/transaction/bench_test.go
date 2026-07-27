package transaction_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/m/internal/transaction"
)

func BenchmarkProjectLockContention(b *testing.B) {
	root := b.TempDir()
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := "bench"
		if err := transaction.AcquireProjectLock(ctx, root, id); err != nil {
			b.Fatal(err)
		}
		if err := transaction.ReleaseProjectLock(root, id); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTransactionCommit(b *testing.B) {
	root := b.TempDir()
	live := filepath.Join(root, "m.lock")
	if err := os.WriteFile(live, []byte("old"), 0o644); err != nil {
		b.Fatal(err)
	}
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		if err := os.WriteFile(live, []byte("old"), 0o644); err != nil {
			b.Fatal(err)
		}
		txn := transaction.NewRunner(root)
		if err := txn.Begin(ctx); err != nil {
			b.Fatal(err)
		}
		stage := txn.StagePath()
		if err := os.WriteFile(filepath.Join(stage, "m.lock"), []byte("new"), 0o644); err != nil {
			b.Fatal(err)
		}
		plan := []transaction.Op{{Kind: transaction.OpRename, Path: "m.lock", Backup: "stage/m.lock"}}
		if err := txn.SetPlan(plan); err != nil {
			b.Fatal(err)
		}
		if err := txn.RecordBackup("m.lock"); err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
		if err := txn.Commit(ctx, nil); err != nil {
			b.Fatal(err)
		}
		b.StopTimer()
		_ = txn.Finish(false)
	}
}

func BenchmarkTransactionRollback(b *testing.B) {
	root := b.TempDir()
	live := filepath.Join(root, "m.lock")
	if err := os.WriteFile(live, []byte("old"), 0o644); err != nil {
		b.Fatal(err)
	}
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		if err := os.WriteFile(live, []byte("old"), 0o644); err != nil {
			b.Fatal(err)
		}
		txn := transaction.NewRunner(root)
		if err := txn.Begin(ctx); err != nil {
			b.Fatal(err)
		}
		if err := txn.RecordBackup("m.lock"); err != nil {
			b.Fatal(err)
		}
		if err := os.WriteFile(live, []byte("mutated"), 0o644); err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
		if _, err := txn.Rollback(ctx); err != nil {
			b.Fatal(err)
		}
		b.StopTimer()
		_ = txn.Finish(false)
	}
}

func BenchmarkTransactionRecoveryScan(b *testing.B) {
	root := b.TempDir()
	live := filepath.Join(root, "m.lock")
	if err := os.WriteFile(live, []byte("old"), 0o644); err != nil {
		b.Fatal(err)
	}
	ctx := context.Background()
	txn := transaction.NewRunner(root)
	if err := txn.Begin(ctx); err != nil {
		b.Fatal(err)
	}
	if err := txn.RecordBackup("m.lock"); err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(live, []byte("partial"), 0o644); err != nil {
		b.Fatal(err)
	}
	if err := txn.SetState(transaction.StateCommitting); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := transaction.RecoverIncomplete(ctx, root); err != nil {
			b.Fatal(err)
		}
		b.StopTimer()
		txn := transaction.NewRunner(root)
		if err := txn.Begin(ctx); err != nil {
			b.Fatal(err)
		}
		if err := txn.RecordBackup("m.lock"); err != nil {
			b.Fatal(err)
		}
		if err := os.WriteFile(live, []byte("partial"), 0o644); err != nil {
			b.Fatal(err)
		}
		if err := txn.SetState(transaction.StateCommitting); err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
	}
}
