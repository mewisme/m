package fsx_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/m/internal/fsx"
)

func TestReleaseDirLockMissingOwnerKeepsDir(t *testing.T) {
	root := t.TempDir()
	lockDir := filepath.Join(root, "lock")
	if err := os.MkdirAll(lockDir, 0o755); err != nil {
		t.Fatal(err)
	}
	result, err := fsx.ReleaseDirLock(lockDir, func([]byte) bool { return true })
	if err != nil {
		t.Fatal(err)
	}
	if result != fsx.ReleaseMissingOwner {
		t.Fatalf("result=%v", result)
	}
	if _, err := os.Stat(lockDir); err != nil {
		t.Fatal("lock dir removed")
	}
}

func TestReleaseDirLockMalformedOwnerKeepsDir(t *testing.T) {
	root := t.TempDir()
	lockDir := filepath.Join(root, "lock")
	if err := os.MkdirAll(lockDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(lockDir, fsx.OwnerFileName), []byte("{bad"), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := fsx.ReleaseDirLock(lockDir, func([]byte) bool { return true })
	if err != nil {
		t.Fatal(err)
	}
	if result != fsx.ReleaseMalformedOwner {
		t.Fatalf("result=%v", result)
	}
	if _, err := os.Stat(lockDir); err != nil {
		t.Fatal("lock dir removed")
	}
}

func TestReleaseDirLockNotOwnerKeepsDir(t *testing.T) {
	root := t.TempDir()
	lockDir := filepath.Join(root, "lock")
	if err := os.MkdirAll(lockDir, 0o755); err != nil {
		t.Fatal(err)
	}
	owner := []byte(`{"lockId":"a","pid":1}` + "\n")
	if err := os.WriteFile(filepath.Join(lockDir, fsx.OwnerFileName), owner, 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := fsx.ReleaseDirLock(lockDir, func(data []byte) bool { return false })
	if err != nil {
		t.Fatal(err)
	}
	if result != fsx.ReleaseNotOwner {
		t.Fatalf("result=%v", result)
	}
	if _, err := os.Stat(lockDir); err != nil {
		t.Fatal("lock dir removed")
	}
}

func TestReleaseDirLockOKRemovesDir(t *testing.T) {
	root := t.TempDir()
	lockDir := filepath.Join(root, "lock")
	if err := os.MkdirAll(lockDir, 0o755); err != nil {
		t.Fatal(err)
	}
	owner := []byte(`{"lockId":"a","pid":1}` + "\n")
	if err := os.WriteFile(filepath.Join(lockDir, fsx.OwnerFileName), owner, 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := fsx.ReleaseDirLock(lockDir, func(data []byte) bool { return true })
	if err != nil {
		t.Fatal(err)
	}
	if result != fsx.ReleaseOK {
		t.Fatalf("result=%v", result)
	}
	if _, err := os.Stat(lockDir); !os.IsNotExist(err) {
		t.Fatal("lock dir should be removed")
	}
}
