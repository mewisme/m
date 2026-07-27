package store_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/m/internal/store"
	"github.com/mewisme/m/internal/testkit"
)

func TestImportDedup(t *testing.T) {
	root := filepath.Join(t.TempDir(), "store")
	ps := store.NewPackageStore(root)
	tgz := filepath.Join(testkit.FixtureDir(t, "registry/v1"), "tarballs", "lodash-4.17.21.tgz")
	integrity := "sha256-758b80171fc185274170cb6db31a08042813d860a47b612d0671122a306b8b63"
	ctx := context.Background()

	k1, err := ps.ImportFromTarball(ctx, tgz, integrity)
	if err != nil {
		t.Fatal(err)
	}
	k2, err := ps.ImportFromTarball(ctx, tgz, integrity)
	if err != nil {
		t.Fatal(err)
	}
	if k1 != k2 {
		t.Fatalf("keys differ: %v %v", k1, k2)
	}
	path := ps.PackagePath(k1)
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyDetectsTamper(t *testing.T) {
	root := filepath.Join(t.TempDir(), "store")
	ps := store.NewPackageStore(root)
	tgz := filepath.Join(testkit.FixtureDir(t, "registry/v1"), "tarballs", "pkg-cli-1.0.0.tgz")
	integrity := "sha256-6ffb2697417ee0f02ad400c8d92c46cfb5889cf84603cd1f797146fde316b5d0"
	ctx := context.Background()
	key, err := ps.ImportFromTarball(ctx, tgz, integrity)
	if err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(ps.PackagePath(key), ".mew-package-integrity")
	if err := os.Chmod(marker, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(marker, []byte("sha256-dead"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ps.VerifyPackage(ctx, key); err == nil {
		t.Fatal("expected verify failure after tamper")
	}
	_, err = ps.ImportFromTarball(ctx, tgz, integrity)
	if err != nil {
		t.Fatal(err)
	}
	if err := ps.VerifyPackage(ctx, key); err != nil {
		t.Fatal(err)
	}
}
