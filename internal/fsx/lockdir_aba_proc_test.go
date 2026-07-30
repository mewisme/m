package fsx_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/mewisme/mew/internal/fsx"
	"github.com/mewisme/mew/internal/jsonfile"
	"github.com/mewisme/mew/internal/store"
	"github.com/mewisme/mew/internal/transaction"
)

func TestTakeoverStaleFileLockABARace(t *testing.T) {
	root := t.TempDir()
	lockPath := filepath.Join(root, "legacy.lock")
	oldOwner := []byte(`{"lockId":"old"}` + "\n")
	if err := os.WriteFile(lockPath, oldOwner, 0o644); err != nil {
		t.Fatal(err)
	}
	obs := fsx.LockObservation{
		LockID:    "old",
		OwnerJSON: append([]byte(nil), oldOwner...),
	}
	tomb := fsx.TombstoneRoot(lockPath)

	if err := os.Remove(lockPath); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lockPath, []byte(`{"lockId":"new"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := fsx.TakeoverStaleFileLock(lockPath, obs, tomb)
	if err == nil {
		t.Fatal("expected takeover failure when lock replaced")
	}
	if !errors.Is(err, os.ErrExist) {
		t.Fatalf("err=%v", err)
	}
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"lockId":"new"}`+"\n" {
		t.Fatalf("live lock replaced: %q", data)
	}
}

func TestTakeoverStaleFileLockTombstonesOnce(t *testing.T) {
	root := t.TempDir()
	lockPath := filepath.Join(root, "legacy.lock")
	tomb := fsx.TombstoneRoot(lockPath)
	owner := []byte(`{"lockId":"stale","pid":999999,"processStart":1}` + "\n")
	if err := os.WriteFile(lockPath, owner, 0o644); err != nil {
		t.Fatal(err)
	}
	obs := fsx.ObservationFromOwner(owner, time.Now(), false)
	if err := fsx.TakeoverStaleFileLock(lockPath, obs, tomb); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatal("stale legacy file lock should be tombstoned")
	}
}

func TestTakeoverStaleDirLockABARace(t *testing.T) {
	root := t.TempDir()
	lockDir := filepath.Join(root, "lock")
	if err := os.Mkdir(lockDir, 0o755); err != nil {
		t.Fatal(err)
	}
	oldOwner := []byte(`{"lockId":"old"}` + "\n")
	if err := os.WriteFile(filepath.Join(lockDir, fsx.OwnerFileName), oldOwner, 0o644); err != nil {
		t.Fatal(err)
	}
	obs := fsx.LockObservation{
		LockID:    "old",
		OwnerJSON: append([]byte(nil), oldOwner...),
	}
	tomb := fsx.TombstoneRoot(lockDir)

	if err := os.RemoveAll(lockDir); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(lockDir, 0o755); err != nil {
		t.Fatal(err)
	}
	newOwner := []byte(`{"lockId":"new"}` + "\n")
	if err := os.WriteFile(filepath.Join(lockDir, fsx.OwnerFileName), newOwner, 0o644); err != nil {
		t.Fatal(err)
	}

	err := fsx.TakeoverStaleDirLock(lockDir, obs, tomb)
	if err == nil {
		t.Fatal("expected takeover failure when lock replaced")
	}
	if !errors.Is(err, os.ErrExist) {
		t.Fatalf("err=%v", err)
	}
	data, err := os.ReadFile(filepath.Join(lockDir, fsx.OwnerFileName))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(newOwner) {
		t.Fatalf("live lock replaced: %q", data)
	}
}

func TestAcquireTakeoverGuardExclusive(t *testing.T) {
	root := t.TempDir()
	lockDir := filepath.Join(root, "lock")
	hold := make(chan struct{})
	go func() {
		release, err := fsx.AcquireTakeoverGuard(context.Background(), lockDir)
		if err != nil {
			t.Error(err)
			return
		}
		close(hold)
		time.Sleep(200 * time.Millisecond)
		release()
	}()
	<-hold
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := fsx.AcquireTakeoverGuard(ctx, lockDir)
	if err == nil {
		t.Fatal("expected guard contention timeout")
	}
}

func TestTakeoverStaleDirLockTombstonesOnce(t *testing.T) {
	root := t.TempDir()
	lockDir := filepath.Join(root, "lock")
	tomb := fsx.TombstoneRoot(lockDir)
	if err := os.MkdirAll(lockDir, 0o755); err != nil {
		t.Fatal(err)
	}
	owner := []byte(`{"lockId":"stale","pid":999999,"processStart":1}` + "\n")
	if err := os.WriteFile(filepath.Join(lockDir, fsx.OwnerFileName), owner, 0o644); err != nil {
		t.Fatal(err)
	}
	obs := fsx.ObservationFromOwner(owner, time.Now(), false)
	if err := fsx.TakeoverStaleDirLock(lockDir, obs, tomb); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(lockDir); !os.IsNotExist(err) {
		t.Fatal("stale lock directory should be tombstoned")
	}
}

func TestTakeoverABAProcWorker(t *testing.T) {
	root := os.Getenv("MEW_TAKEOVER_ABA_ROOT")
	lockKind := os.Getenv("MEW_TAKEOVER_ABA_KIND")
	if root == "" || lockKind == "" {
		t.Skip("takeover ABA worker subprocess only")
	}
	signal := filepath.Join(root, "continue")
	_ = os.Remove(signal)

	var lockDir string
	switch lockKind {
	case "project":
		lockDir = transaction.LockPath(root)
	case "import":
		lockDir = filepath.Join(root, ".locks", "sha256", "deadbeef")
	case "index":
		lockDir = filepath.Join(root, ".locks", "index")
	case "legacy-file":
		lockPath := filepath.Join(root, ".locks", "sha256", "deadbeef.lock")
		staleOwner, err := os.ReadFile(lockPath)
		if err != nil {
			t.Fatal(err)
		}
		obs := fsx.ObservationFromOwner(staleOwner, time.Now(), false)
		tomb := fsx.TombstoneRoot(filepath.Join(root, ".locks", "sha256", "deadbeef"))
		err = fsx.TakeoverStaleFileLock(lockPath, obs, tomb)
		result := filepath.Join(root, "result-"+lockKind)
		if err == nil {
			_ = os.WriteFile(result, []byte("unexpected-success"), 0o644)
			return
		}
		if !errors.Is(err, os.ErrExist) {
			_ = os.WriteFile(result, []byte(err.Error()), 0o644)
			return
		}
		_ = os.WriteFile(result, []byte("aba-blocked"), 0o644)
		return
	default:
		t.Fatalf("unknown kind %q", lockKind)
	}

	staleOwner, err := os.ReadFile(filepath.Join(lockDir, fsx.OwnerFileName))
	if err != nil {
		t.Fatal(err)
	}

	obs := fsx.ObservationFromOwner(staleOwner, time.Now(), false)
	tomb := fsx.TombstoneRoot(lockDir)
	err = fsx.TakeoverStaleDirLock(lockDir, obs, tomb)
	result := filepath.Join(root, "result-"+lockKind)
	if err == nil {
		_ = os.WriteFile(result, []byte("unexpected-success"), 0o644)
		return
	}
	if !errors.Is(err, os.ErrExist) {
		_ = os.WriteFile(result, []byte(err.Error()), 0o644)
		return
	}
	_ = os.WriteFile(result, []byte("aba-blocked"), 0o644)
}

func runTakeoverABAProcTest(t *testing.T, kind string, setup func(root string) (lockDir string, staleOwner []byte)) {
	t.Helper()
	if testing.Short() {
		t.Skip("ABA proc test")
	}
	root := t.TempDir()
	lockDir, staleOwner := setup(root)
	if err := os.MkdirAll(filepath.Dir(lockDir), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(lockDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(lockDir, fsx.OwnerFileName), staleOwner, 0o644); err != nil {
		t.Fatal(err)
	}

	signal := filepath.Join(root, "continue")
	ready := filepath.Join(root, "worker-ready")
	_ = os.Remove(signal)
	_ = os.Remove(ready)
	exe, err := exec.LookPath(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe, "-test.run=^TestTakeoverABAProcWorker$", "-test.count=1")
	cmd.Env = append(os.Environ(),
		"MEW_TAKEOVER_ABA_ROOT="+root,
		"MEW_TAKEOVER_ABA_KIND="+kind,
		"MEW_LOCK_TAKEOVER_PAUSE=pre-rename",
		"MEW_LOCK_TAKEOVER_SIGNAL="+signal,
		"MEW_LOCK_TAKEOVER_READY="+ready,
	)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(ready); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err := os.Stat(ready); err != nil {
		t.Fatal("worker did not reach pre-rename pause")
	}

	_ = os.RemoveAll(lockDir)
	if err := os.MkdirAll(lockDir, 0o755); err != nil {
		t.Fatal(err)
	}
	newOwner := []byte(`{"schemaVersion":2,"lockId":"live","pid":1,"processStart":2}` + "\n")
	if kind == "project" {
		doc := transaction.LockDocument{
			SchemaVersion: 3,
			LockID:        "live",
			PID:           1,
			ProcessStart:  2,
			TxnID:         "live-txn",
			ProjectRoot:   root,
			CreatedAt:     time.Now().UTC(),
		}
		newOwner, err = jsonfile.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(lockDir, fsx.OwnerFileName), newOwner, 0o644); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(signal, []byte("go"), 0o644)

	if err := cmd.Wait(); err != nil {
		t.Fatalf("worker exit: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, "result-"+kind))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "aba-blocked" {
		t.Fatalf("result=%q", data)
	}
	live, err := os.ReadFile(filepath.Join(lockDir, fsx.OwnerFileName))
	if err != nil {
		t.Fatal(err)
	}
	if string(live) != string(newOwner) {
		t.Fatalf("live lock mutated: %q", live)
	}
}

func runTakeoverABAFileProcTest(t *testing.T, kind string, setup func(root string) (lockPath string, staleOwner []byte)) {
	t.Helper()
	if testing.Short() {
		t.Skip("ABA proc test")
	}
	root := t.TempDir()
	lockPath, staleOwner := setup(root)
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lockPath, staleOwner, 0o644); err != nil {
		t.Fatal(err)
	}

	signal := filepath.Join(root, "continue")
	ready := filepath.Join(root, "worker-ready")
	_ = os.Remove(signal)
	_ = os.Remove(ready)
	exe, err := exec.LookPath(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe, "-test.run=^TestTakeoverABAProcWorker$", "-test.count=1")
	cmd.Env = append(os.Environ(),
		"MEW_TAKEOVER_ABA_ROOT="+root,
		"MEW_TAKEOVER_ABA_KIND="+kind,
		"MEW_LOCK_TAKEOVER_PAUSE=pre-rename",
		"MEW_LOCK_TAKEOVER_SIGNAL="+signal,
		"MEW_LOCK_TAKEOVER_READY="+ready,
	)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(ready); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err := os.Stat(ready); err != nil {
		t.Fatal("worker did not reach pre-rename pause")
	}

	_ = os.Remove(lockPath)
	newOwner := []byte(`{"schemaVersion":2,"lockId":"live","pid":1,"processStart":2}` + "\n")
	if err := os.WriteFile(lockPath, newOwner, 0o644); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(signal, []byte("go"), 0o644)

	if err := cmd.Wait(); err != nil {
		t.Fatalf("worker exit: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, "result-"+kind))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "aba-blocked" {
		t.Fatalf("result=%q", data)
	}
	live, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(live) != string(newOwner) {
		t.Fatalf("live lock mutated: %q", live)
	}
}

func TestTakeoverABAProcLegacyFileLock(t *testing.T) {
	runTakeoverABAFileProcTest(t, "legacy-file", func(root string) (string, []byte) {
		lockPath := filepath.Join(root, ".locks", "sha256", "deadbeef.lock")
		doc := store.ImportLockDocument{
			SchemaVersion: 2,
			LockID:        "stale",
			PID:           999999,
			ProcessStart:  1,
			PackageKey:    "sha256/deadbeef",
			CreatedAt:     time.Now().UTC(),
		}
		raw, _ := jsonfile.Marshal(doc)
		return lockPath, raw
	})
}

func TestTakeoverABAProcProjectLock(t *testing.T) {
	runTakeoverABAProcTest(t, "project", func(root string) (string, []byte) {
		lockDir := transaction.LockPath(root)
		doc := transaction.LockDocument{
			SchemaVersion: 3,
			LockID:        "stale",
			PID:           999999,
			ProcessStart:  1,
			TxnID:         "stale-txn",
			ProjectRoot:   root,
			CreatedAt:     time.Now().UTC(),
		}
		raw, _ := jsonfile.Marshal(doc)
		return lockDir, raw
	})
}

func TestTakeoverABAProcImportLock(t *testing.T) {
	runTakeoverABAProcTest(t, "import", func(root string) (string, []byte) {
		lockDir := filepath.Join(root, ".locks", "sha256", "deadbeef")
		doc := store.ImportLockDocument{
			SchemaVersion: 2,
			LockID:        "stale",
			PID:           999999,
			ProcessStart:  1,
			PackageKey:    "sha256/deadbeef",
			CreatedAt:     time.Now().UTC(),
		}
		raw, _ := jsonfile.Marshal(doc)
		return lockDir, raw
	})
}

func TestTakeoverABAProcIndexLock(t *testing.T) {
	runTakeoverABAProcTest(t, "index", func(root string) (string, []byte) {
		lockDir := filepath.Join(root, ".locks", "index")
		doc := store.IndexLockDocument{
			SchemaVersion: 2,
			LockID:        "stale",
			PID:           999999,
			ProcessStart:  1,
			CreatedAt:     time.Now().UTC(),
		}
		raw, _ := jsonfile.Marshal(doc)
		return lockDir, raw
	})
}
