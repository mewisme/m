package fsx_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/fsx"
)

func TestGuardAncestorsRejectsSymlinkComponent(t *testing.T) {
	root := t.TempDir()
	real := filepath.Join(root, "real")
	mew := filepath.Join(root, "proj", ".mew")
	if err := os.MkdirAll(real, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(mew), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "proj", ".mew")
	_ = os.RemoveAll(link)
	if err := os.Symlink(real, link); err != nil {
		t.Skip("symlink not supported:", err)
	}
	target := filepath.Join(root, "proj", ".mew", "txn", "abc")
	if err := fsx.GuardAncestors(filepath.Join(root, "proj"), target); err == nil {
		t.Fatal("expected symlink rejection")
	}
}

func TestGuardAncestorsAllowsNormalTree(t *testing.T) {
	root := t.TempDir()
	mew := filepath.Join(root, ".mew", "txn", "abc")
	if err := os.MkdirAll(mew, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := fsx.GuardAncestors(root, mew); err != nil {
		t.Fatal(err)
	}
}

func TestGuardAncestorsRejectsProjectRootSymlink(t *testing.T) {
	base := t.TempDir()
	real := filepath.Join(base, "real")
	if err := os.MkdirAll(real, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "proj")
	if err := os.Symlink(real, link); err != nil {
		t.Skip("symlink not supported:", err)
	}
	target := filepath.Join(link, ".mew", "store-manifest.json")
	err := fsx.GuardAncestors(link, target)
	if err == nil {
		t.Fatal("expected project root symlink rejection")
	}
	if apperr.CodeOf(err) != apperr.Transaction {
		t.Fatalf("code=%s err=%v", apperr.CodeOf(err), err)
	}
}

func TestGuardAncestorsNodeModulesSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	nm := filepath.Join(root, "node_modules", "pkg")
	if err := os.MkdirAll(filepath.Dir(nm), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "package.json"), []byte(`{"name":"evil"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, nm); err != nil {
		t.Skip("symlink not supported:", err)
	}
	if err := fsx.GuardAncestors(root, filepath.Join(nm, "package.json")); err == nil {
		t.Fatal("expected node_modules symlink rejection")
	}
}

func TestGuardAncestorsSnapshotsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	snap := filepath.Join(root, ".mew", "snapshots", "s1")
	if err := os.MkdirAll(filepath.Dir(snap), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, snap); err != nil {
		t.Skip("symlink not supported:", err)
	}
	if err := fsx.GuardAncestors(root, filepath.Join(snap, "meta.json")); err == nil {
		t.Fatal("expected snapshots symlink rejection")
	}
}

func TestRequiresAncestorGuard(t *testing.T) {
	cases := map[string]bool{
		".":                           true,
		".mew/txn/id":                 true,
		"node_modules/pkg":            true,
		"packages/a/node_modules/pkg": true,
		".mew/snapshots/s1":           true,
		"package.json":                false,
		"src/index.ts":                false,
		".mew-old/node_modules":       false,
	}
	for rel, want := range cases {
		if got := fsx.RequiresAncestorGuard(rel); got != want {
			t.Fatalf("%q: got %v want %v", rel, got, want)
		}
	}
}
