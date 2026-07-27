package store_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mewisme/m/internal/store"
	"github.com/mewisme/m/internal/testkit"
)

func TestTreeManifestDetectsFileTamper(t *testing.T) {
	root := filepath.Join(t.TempDir(), "store")
	ps := store.NewPackageStore(root)
	tgz := filepath.Join(testkit.FixtureDir(t, "registry/v1"), "tarballs", "pkg-cli-1.0.0.tgz")
	integrity := "sha256-6ffb2697417ee0f02ad400c8d92c46cfb5889cf84603cd1f797146fde316b5d0"
	ctx := context.Background()
	key, err := ps.ImportFromTarball(ctx, tgz, integrity)
	if err != nil {
		t.Fatal(err)
	}
	pkgDir := ps.PackagePath(key)
	target := filepath.Join(pkgDir, "package.json")
	if err := os.Chmod(target, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ps.VerifyPackage(ctx, key); err == nil {
		t.Fatal("expected verify failure after file tamper")
	}
}

func TestTreeManifestDetectsModeDrift(t *testing.T) {
	root := filepath.Join(t.TempDir(), "store")
	ps := store.NewPackageStore(root)
	tgz := filepath.Join(testkit.FixtureDir(t, "registry/v1"), "tarballs", "lodash-4.17.21.tgz")
	integrity := "sha256-758b80171fc185274170cb6db31a08042813d860a47b612d0671122a306b8b63"
	ctx := context.Background()
	key, err := ps.ImportFromTarball(ctx, tgz, integrity)
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(ps.PackagePath(key), "package.json")
	if err := os.Chmod(target, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ps.VerifyPackage(ctx, key); err == nil {
		t.Fatal("expected mode drift failure")
	}
}

func TestImportRepairsCorruption(t *testing.T) {
	root := filepath.Join(t.TempDir(), "store")
	ps := store.NewPackageStore(root)
	tgz := filepath.Join(testkit.FixtureDir(t, "registry/v1"), "tarballs", "pkg-cli-1.0.0.tgz")
	integrity := "sha256-6ffb2697417ee0f02ad400c8d92c46cfb5889cf84603cd1f797146fde316b5d0"
	ctx := context.Background()
	key, err := ps.ImportFromTarball(ctx, tgz, integrity)
	if err != nil {
		t.Fatal(err)
	}
	pkgDir := ps.PackagePath(key)
	if err := os.Chmod(filepath.Join(pkgDir, "package.json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ps.ImportFromTarball(ctx, tgz, integrity); err != nil {
		t.Fatal(err)
	}
	if err := ps.VerifyPackage(ctx, key); err != nil {
		t.Fatal(err)
	}
	quarantine := filepath.Join(root, ".quarantine", "sha256", key.Hex)
	if _, err := os.Stat(quarantine); err != nil {
		t.Fatalf("expected quarantine dir: %v", err)
	}
}

func TestCleanupStaleStaging(t *testing.T) {
	root := filepath.Join(t.TempDir(), "store")
	ps := store.NewPackageStore(root)
	stale := filepath.Join(root, ".staging", "old")
	if err := os.MkdirAll(stale, 0o755); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(stale, old, old); err != nil {
		t.Fatal(err)
	}
	fresh := filepath.Join(root, ".staging", "new")
	if err := os.MkdirAll(fresh, 0o755); err != nil {
		t.Fatal(err)
	}
	n, err := ps.CleanupStaleStaging(time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("removed=%d want 1", n)
	}
	if _, err := os.Stat(fresh); err != nil {
		t.Fatal("fresh staging should remain")
	}
}

func TestPruneSkipsImportLock(t *testing.T) {
	root := filepath.Join(t.TempDir(), "store")
	ps := store.NewPackageStore(root)
	tgz := filepath.Join(testkit.FixtureDir(t, "registry/v1"), "tarballs", "lodash-4.17.21.tgz")
	integrity := "sha256-758b80171fc185274170cb6db31a08042813d860a47b612d0671122a306b8b63"
	ctx := context.Background()
	key, err := ps.ImportFromTarball(ctx, tgz, integrity)
	if err != nil {
		t.Fatal(err)
	}
	lock := filepath.Join(ps.PackagePath(key), ".import.lock")
	if err := os.WriteFile(lock, []byte("1"), 0o644); err != nil {
		t.Fatal(err)
	}
	candidates, err := store.PruneCandidates(ps, map[string]struct{}{})
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 0 {
		t.Fatalf("expected no prune candidates with import lock, got %d", len(candidates))
	}
}

func TestConcurrentImportDedup(t *testing.T) {
	root := filepath.Join(t.TempDir(), "store")
	ps := store.NewPackageStore(root)
	tgz := filepath.Join(testkit.FixtureDir(t, "registry/v1"), "tarballs", "lodash-4.17.21.tgz")
	integrity := "sha256-758b80171fc185274170cb6db31a08042813d860a47b612d0671122a306b8b63"
	ctx := context.Background()
	errCh := make(chan error, 4)
	for i := 0; i < 4; i++ {
		go func() {
			_, err := ps.ImportFromTarball(ctx, tgz, integrity)
			errCh <- err
		}()
	}
	for i := 0; i < 4; i++ {
		if err := <-errCh; err != nil {
			t.Fatal(err)
		}
	}
	keys, err := ps.ListPackageKeys()
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
}
