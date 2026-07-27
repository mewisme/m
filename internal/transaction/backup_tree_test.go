package transaction_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/mewisme/m/internal/transaction"
)

func TestBackupTreeFileSymlinkRoundTrip(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation may require elevation on Windows")
	}
	root := t.TempDir()
	target := filepath.Join(root, "real.txt")
	if err := os.WriteFile(target, []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Skip(err)
	}
	backupRoot := filepath.Join(root, "backup")
	live := filepath.Join(root, "live")
	if err := os.Symlink(target, live); err != nil {
		t.Fatal(err)
	}

	txn := transaction.NewRunner(root)
	ctx := context.Background()
	if err := txn.Begin(ctx); err != nil {
		t.Fatal(err)
	}
	if err := txn.RecordBackup("live"); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(live, []byte("mutated"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := txn.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	got, err := os.Readlink(live)
	if err != nil {
		t.Fatal(err)
	}
	if got != target {
		t.Fatalf("symlink target=%q want %q", got, target)
	}
	_ = backupRoot
}

func TestBackupTreeDirSymlinkNoRecurse(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation may require elevation on Windows")
	}
	root := t.TempDir()
	outside := filepath.Join(root, "outside")
	if err := os.MkdirAll(filepath.Join(outside, "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "pkg", "index.js"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	nm := filepath.Join(root, "node_modules")
	if err := os.MkdirAll(nm, 0o755); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(nm, "foo")
	if err := os.Symlink(outside, alias); err != nil {
		t.Skip(err)
	}

	txn := transaction.NewRunner(root)
	ctx := context.Background()
	if err := txn.Begin(ctx); err != nil {
		t.Fatal(err)
	}
	if err := txn.RecordBackup("node_modules"); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(outside); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(nm); err != nil {
		t.Fatal(err)
	}
	if _, err := txn.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Readlink(filepath.Join(nm, "foo")); err != nil {
		t.Fatalf("dir symlink not preserved: %v", err)
	}
	if _, err := os.Stat(filepath.Join(nm, "foo", "pkg", "index.js")); !os.IsNotExist(err) {
		t.Fatal("backup followed symlinked directory")
	}
}

func TestBackupTreeNestedScopedPackages(t *testing.T) {
	root := t.TempDir()
	scope := filepath.Join(root, "node_modules", ".pnpm", "pkg@1.0.0", "node_modules", "pkg")
	if err := os.MkdirAll(scope, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scope, "index.js"), []byte("v1"), 0o644); err != nil {
		t.Fatal(err)
	}

	txn := transaction.NewRunner(root)
	ctx := context.Background()
	if err := txn.Begin(ctx); err != nil {
		t.Fatal(err)
	}
	if err := txn.RecordBackup("node_modules"); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(root, "node_modules")); err != nil {
		t.Fatal(err)
	}
	if _, err := txn.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(scope, "index.js"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "v1" {
		t.Fatalf("nested restore failed: %q", data)
	}
}

func TestBackupTreeUnsupportedSpecialFileFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fifo unsupported test is unix-only")
	}
	root := t.TempDir()
	fifo := filepath.Join(root, "pipe")
	if err := syscallMkfifo(fifo, 0o600); err != nil {
		t.Skip(err)
	}
	txn := transaction.NewRunner(root)
	ctx := context.Background()
	if err := txn.Begin(ctx); err != nil {
		t.Fatal(err)
	}
	err := txn.RecordBackup("pipe")
	if err == nil {
		t.Fatal("expected backup failure for fifo")
	}
	if !strings.Contains(err.Error(), "pipe") && !strings.Contains(err.Error(), "supported") {
		t.Fatalf("unexpected error: %v", err)
	}
}
