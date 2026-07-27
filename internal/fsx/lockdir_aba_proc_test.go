package fsx_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/m/internal/fsx"
)

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
