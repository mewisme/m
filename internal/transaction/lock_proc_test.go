package transaction_test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/fsx"
	"github.com/mewisme/mew/internal/jsonfile"
	"github.com/mewisme/mew/internal/transaction"
)

func lockOwnerPath(root string) string {
	return filepath.Join(transaction.LockPath(root), fsx.OwnerFileName)
}

func TestLockContentionWorker(t *testing.T) {
	root := os.Getenv("MEW_LOCK_TEST_ROOT")
	if root == "" {
		t.Skip("lock contention worker subprocess only")
	}
	slot := os.Getenv("MEW_LOCK_TEST_SLOT")
	acquired := filepath.Join(root, "acquired-"+slot)
	waitCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	err := transaction.AcquireProjectLock(waitCtx, root, "txn-"+slot)
	if err != nil {
		_ = os.WriteFile(acquired, []byte("blocked"), 0o644)
		return
	}
	_ = os.WriteFile(acquired, []byte("ok"), 0o644)
	time.Sleep(1500 * time.Millisecond)
	_ = transaction.ReleaseProjectLock(root, "txn-"+slot)
}

func TestProjectLockContention20Processes(t *testing.T) {
	if testing.Short() {
		t.Skip("20-process lock contention")
	}
	root := t.TempDir()
	const workers = 20
	exe, err := exec.LookPath(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	type outcome struct {
		slot string
		ok   bool
	}
	out := make(chan outcome, workers)
	for i := 0; i < workers; i++ {
		i := i
		go func() {
			slot := strconv.Itoa(i)
			cmd := exec.Command(exe, "-test.run=^TestLockContentionWorker$", "-test.count=1")
			cmd.Env = append(os.Environ(),
				"MEW_LOCK_TEST_ROOT="+root,
				"MEW_LOCK_TEST_SLOT="+slot,
			)
			runErr := cmd.Run()
			data, _ := os.ReadFile(filepath.Join(root, "acquired-"+slot))
			out <- outcome{slot: slot, ok: runErr == nil && string(data) == "ok"}
		}()
	}
	var winners int
	for i := 0; i < workers; i++ {
		if (<-out).ok {
			winners++
		}
	}
	if winners != 1 {
		t.Fatalf("expected exactly one lock winner, got %d", winners)
	}
	time.Sleep(2 * time.Second)
}

func TestProjectLockStaleProcessIdentity(t *testing.T) {
	root := t.TempDir()
	lockDir := transaction.LockPath(root)
	if err := os.MkdirAll(filepath.Dir(lockDir), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(lockDir, 0o755); err != nil {
		t.Fatal(err)
	}
	doc := transaction.LockDocument{
		SchemaVersion: 3,
		LockID:        "stale-lock",
		PID:           999999,
		ProcessStart:  1,
		TxnID:         "stale",
		CreatedAt:     time.Now().UTC(),
		ProjectRoot:   root,
	}
	raw, err := jsonfile.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lockOwnerPath(root), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := transaction.AcquireProjectLock(ctx, root, "fresh"); err != nil {
		t.Fatal(err)
	}
	_ = transaction.ReleaseProjectLock(root, "fresh")
}

func TestProjectLockLiveNotStolen(t *testing.T) {
	root := t.TempDir()
	ctx := context.Background()
	if err := transaction.AcquireProjectLock(ctx, root, "owner"); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = transaction.ReleaseProjectLock(root, "owner") }()

	held := make(chan error, 1)
	go func() {
		waitCtx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()
		held <- transaction.AcquireProjectLock(waitCtx, root, "intruder")
	}()
	err := <-held
	if err == nil {
		t.Fatal("expected lock contention error")
	}
	if apperr.CodeOf(err) != apperr.Cancelled && apperr.CodeOf(err) != apperr.Transaction {
		t.Fatalf("unexpected error code %s: %v", apperr.CodeOf(err), err)
	}
}

func TestProjectLockOwnerMismatchOnRelease(t *testing.T) {
	root := t.TempDir()
	ctx := context.Background()
	if err := transaction.AcquireProjectLock(ctx, root, "mine"); err != nil {
		t.Fatal(err)
	}
	if err := transaction.ReleaseProjectLock(root, "other"); err == nil {
		t.Fatal("expected not-owner release error")
	}
	if _, err := os.Stat(transaction.LockPath(root)); err != nil {
		t.Fatal("lock removed with wrong txn id")
	}
	_ = transaction.ReleaseProjectLock(root, "mine")
}

func TestProjectLockContextCancelDuringWait(t *testing.T) {
	root := t.TempDir()
	ctx := context.Background()
	if err := transaction.AcquireProjectLock(ctx, root, "holder"); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = transaction.ReleaseProjectLock(root, "holder") }()

	waitCtx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	err := transaction.AcquireProjectLock(waitCtx, root, "waiter")
	if err == nil {
		t.Fatal("expected cancel")
	}
	code := apperr.CodeOf(err)
	if code != apperr.Cancelled && code != apperr.Transaction {
		t.Fatalf("expected ERR_M_CANCELLED or contention, got %s", code)
	}
}

func TestProjectLockPIDReuseSimulation(t *testing.T) {
	root := t.TempDir()
	lockDir := transaction.LockPath(root)
	if err := os.MkdirAll(filepath.Dir(lockDir), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(lockDir, 0o755); err != nil {
		t.Fatal(err)
	}
	pid := os.Getpid()
	doc := transaction.LockDocument{
		SchemaVersion: 3,
		LockID:        "reused",
		PID:           pid,
		ProcessStart:  1,
		TxnID:         "reused",
		CreatedAt:     time.Now().UTC(),
		ProjectRoot:   root,
	}
	raw, err := jsonfile.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lockOwnerPath(root), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := transaction.AcquireProjectLock(ctx, root, "new"); err != nil {
		t.Fatalf("stale pid-reuse lock should be replaced: %v", err)
	}
	_ = transaction.ReleaseProjectLock(root, "new")
}

func TestProjectLockDirExclusiveCreate(t *testing.T) {
	root := t.TempDir()
	lockDir := transaction.LockPath(root)
	if err := os.MkdirAll(filepath.Dir(lockDir), 0o755); err != nil {
		t.Fatal(err)
	}
	var created int32
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 100; i++ {
			if err := os.Mkdir(lockDir, 0o755); err == nil {
				created++
				return
			}
			time.Sleep(time.Millisecond)
		}
	}()
	<-done
	if created != 1 {
		t.Fatalf("Mkdir should allow one creator, got %d", created)
	}
	_ = os.RemoveAll(lockDir)
}

func TestLockDocumentV3Fields(t *testing.T) {
	root := t.TempDir()
	ctx := context.Background()
	txnID := "field-check"
	if err := transaction.AcquireProjectLock(ctx, root, txnID); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(lockOwnerPath(root))
	if err != nil {
		t.Fatal(err)
	}
	var doc transaction.LockDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.SchemaVersion < 3 {
		t.Fatalf("schemaVersion=%d", doc.SchemaVersion)
	}
	if doc.LockID == "" {
		t.Fatal("missing lockId")
	}
	if doc.TxnID != txnID || doc.PID != os.Getpid() || doc.ProcessStart == 0 {
		t.Fatalf("unexpected lock doc: %+v", doc)
	}
	if doc.ProjectRoot == "" || doc.CreatedAt.IsZero() {
		t.Fatalf("missing metadata: %+v", doc)
	}
	_ = transaction.ReleaseProjectLock(root, txnID)
}

func TestProjectLockMalformedGracePeriod(t *testing.T) {
	root := t.TempDir()
	lockDir := transaction.LockPath(root)
	if err := os.MkdirAll(lockDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lockOwnerPath(root), []byte("{trunc"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	err := transaction.AcquireProjectLock(ctx, root, "waiter")
	if err == nil {
		t.Fatal("expected wait during grace period")
	}
}

func TestProjectLockStaleTakeoverTombstone(t *testing.T) {
	root := t.TempDir()
	lockDir := transaction.LockPath(root)
	if err := os.MkdirAll(lockDir, 0o755); err != nil {
		t.Fatal(err)
	}
	doc := transaction.LockDocument{
		SchemaVersion: 3,
		LockID:        "stale-id",
		PID:           999999,
		ProcessStart:  1,
		TxnID:         "stale",
		CreatedAt:     time.Now().UTC().Add(-time.Minute),
		ProjectRoot:   root,
	}
	raw, _ := jsonfile.Marshal(doc)
	if err := os.WriteFile(lockOwnerPath(root), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := transaction.AcquireProjectLock(ctx, root, "fresh"); err != nil {
		t.Fatal(err)
	}
	tombRoot := fsx.TombstoneRoot(lockDir)
	if _, err := os.Stat(filepath.Join(tombRoot, ".stale")); err != nil {
		t.Fatalf("expected tombstone dir: %v", err)
	}
	_ = transaction.ReleaseProjectLock(root, "fresh")
}

func TestRecoveryTakeoverReplacesDeadHolder(t *testing.T) {
	root := t.TempDir()
	lockDir := transaction.LockPath(root)
	if err := os.MkdirAll(lockDir, 0o755); err != nil {
		t.Fatal(err)
	}
	doc := transaction.LockDocument{
		SchemaVersion: 3,
		LockID:        "dead",
		PID:           999999,
		ProcessStart:  1,
		TxnID:         "dead-txn",
		CreatedAt:     time.Now().UTC().Add(-time.Minute),
		ProjectRoot:   root,
	}
	raw, _ := jsonfile.Marshal(doc)
	if err := os.WriteFile(lockOwnerPath(root), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := transaction.TakeoverProjectLock(ctx, root, "recovery-txn"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(lockOwnerPath(root))
	if err != nil {
		t.Fatal(err)
	}
	var got transaction.LockDocument
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.TxnID != "recovery-txn" || got.PID != os.Getpid() {
		t.Fatalf("unexpected takeover doc: %+v", got)
	}
	_ = transaction.ReleaseProjectLock(root, "recovery-txn")
}
