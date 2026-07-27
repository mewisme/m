package store

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/contentid"
	"github.com/mewisme/m/internal/fsx"
	"github.com/mewisme/m/internal/testkit"
)

func TestImportFromTarballSurfacesReleaseFailure(t *testing.T) {
	root := filepath.Join(t.TempDir(), "store")
	ps := NewPackageStore(root)
	tgz := filepath.Join(testkit.FixtureDir(t, "registry/v1"), "tarballs", "lodash-4.17.21.tgz")
	integrity := "sha256-758b80171fc185274170cb6db31a08042813d860a47b612d0671122a306b8b63"
	key, err := PackageKeyFromIntegrity(integrity)
	if err != nil {
		t.Fatal(err)
	}

	importLockReleaseTestHook = func(lockDir string) error {
		return apperr.New(apperr.Store, "store.import.lock.release", lockDir, "lock not released: not owner")
	}
	t.Cleanup(func() { importLockReleaseTestHook = nil })

	result, err := ps.ImportFromTarball(context.Background(), tgz, mustParseSRI(t, integrity))
	if err != nil {
		t.Fatal(err)
	}
	if result.Key != key {
		t.Fatalf("key=%v want %v", result.Key, key)
	}
	if len(result.CleanupWarnings) == 0 {
		t.Fatal("expected cleanup warning for import lock release failure")
	}
	if !strings.Contains(result.CleanupWarnings[0], "not owner") {
		t.Fatalf("warning=%q", result.CleanupWarnings[0])
	}
	if err := ps.VerifyPackage(context.Background(), result.Key); err != nil {
		t.Fatalf("package should remain valid: %v", err)
	}
	if !HasImportLock(ps, key) {
		t.Fatal("import lock should remain live when release fails")
	}
}

func TestReconcileIndexSurfacesIndexLockReleaseFailure(t *testing.T) {
	root := filepath.Join(t.TempDir(), "store")
	rep := &captureReporter{}
	ps := NewPackageStore(root)
	ps.Reporter = rep
	tgz := filepath.Join(testkit.FixtureDir(t, "registry/v1"), "tarballs", "lodash-4.17.21.tgz")
	integrity := "sha256-758b80171fc185274170cb6db31a08042813d860a47b612d0671122a306b8b63"
	if _, err := importIntegrity(context.Background(), ps, tgz, integrity); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(indexPathForTest(root)); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}

	indexLockReleaseTestHook = func(lockDir string) error {
		return apperr.New(apperr.Store, "store.index.lock.release", lockDir, "lock not released: not owner")
	}
	t.Cleanup(func() { indexLockReleaseTestHook = nil })

	result, err := ps.ReconcileIndex()
	if err != nil {
		t.Fatal(err)
	}
	if result.Added != 1 {
		t.Fatalf("added=%d", result.Added)
	}
	if !strings.Contains(rep.buf.String(), "warning: store index lock release failed") {
		t.Fatalf("expected maintenance warning, got %q", rep.buf.String())
	}
}

func TestImportLockReleaseNotOwner(t *testing.T) {
	root := t.TempDir()
	key := PackageKey{Algo: "sha256", Hex: "abc"}
	release, err := acquireImportLock(context.Background(), root, key)
	if err != nil {
		t.Fatal(err)
	}
	lockDir := externalImportLockPath(root, key)
	if err := os.WriteFile(filepath.Join(lockDir, fsx.OwnerFileName), []byte(`{"lockId":"other"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := release(); err == nil {
		t.Fatal("expected release error")
	} else if !strings.Contains(err.Error(), "not owner") {
		t.Fatalf("err=%v", err)
	}
	if _, err := os.Stat(lockDir); err != nil {
		t.Fatal("lock dir should remain when release is not owner")
	}
}

func mustParseSRI(t *testing.T, sri string) contentid.Identity {
	t.Helper()
	id, err := contentid.ParseSRI(sri)
	if err != nil {
		t.Fatal(err)
	}
	return id
}
