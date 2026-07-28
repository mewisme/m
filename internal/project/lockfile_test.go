package project_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/project"
)

func TestLockFilename(t *testing.T) {
	if project.LockFilename(project.IdentityMew) != "m.lock" {
		t.Fatal("mew lock filename")
	}
	if project.LockFilename(project.IdentityNub) != "nub.lock" {
		t.Fatal("nub lock filename")
	}
	if project.LockFilename(project.IdentityPNPM) != "pnpm-lock.yaml" {
		t.Fatal("pnpm lock filename")
	}
}

func TestIncumbentLockPathAndReadBytes(t *testing.T) {
	root := filepath.Join("..", "..", "fixtures", "locks", "nub", "v1-basic")
	path, ok := project.IncumbentLockPath(root, project.IdentityNub)
	if !ok {
		t.Fatal("expected nub.lock")
	}
	if filepath.Base(path) != "nub.lock" {
		t.Fatalf("path=%s", path)
	}
	data, err := project.ReadLockfileBytes(root, project.IdentityNub)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("expected lock bytes")
	}
}

func TestDetectIncumbentLock(t *testing.T) {
	root := filepath.Join("..", "..", "fixtures", "locks", "pnpm", "v9")
	id, path, ok := project.DetectIncumbentLock(root)
	if !ok {
		t.Fatal("expected lock detection")
	}
	if id != project.IdentityPNPM {
		t.Fatalf("id=%s", id)
	}
	if filepath.Base(path) != "pnpm-lock.yaml" {
		t.Fatalf("path=%s", path)
	}
}

func TestDetectIncumbentLockAmbiguous(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "m.lock"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "nub.lock"), []byte("lockfileVersion: '9.0'"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, ok := project.DetectIncumbentLock(dir); ok {
		t.Fatal("multiple lockfiles must not resolve")
	}
}
