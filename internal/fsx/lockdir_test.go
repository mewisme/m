package fsx_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mewisme/mew/internal/fsx"
)

func TestDirLockExclusive(t *testing.T) {
	root := t.TempDir()
	lockDir := filepath.Join(root, "lock")
	owner := []byte(`{"id":"a"}` + "\n")
	ctx := context.Background()
	release, err := fsx.AcquireDirLock(ctx, lockDir, owner, fsx.DirLockOptions{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	_, err = fsx.AcquireDirLock(ctx, lockDir, []byte(`{"id":"b"}`+"\n"), fsx.DirLockOptions{MaxWait: 50 * time.Millisecond}, func([]byte, time.Time) bool { return false }, nil)
	if err == nil {
		t.Fatal("expected contention")
	}
}

func TestDirLockMalformedGrace(t *testing.T) {
	root := t.TempDir()
	lockDir := filepath.Join(root, "lock")
	if err := os.MkdirAll(lockDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(lockDir, fsx.OwnerFileName), []byte("{bad"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := fsx.AcquireDirLock(ctx, lockDir, []byte(`{"id":"x"}`+"\n"), fsx.DirLockOptions{}, func(data []byte, mod time.Time) bool {
		if len(data) == 0 {
			return time.Since(mod) >= fsx.DefaultLockGrace
		}
		return false
	}, nil)
	if err == nil {
		t.Fatal("expected wait during grace")
	}
}

func TestWriteAtomicPreservesPrior(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "head")
	if err := fsx.WriteAtomic(path, []byte("gen=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteAtomic(path, []byte("gen=2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "gen=2\n" {
		t.Fatalf("got %q", data)
	}
}
