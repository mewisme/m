//go:build windows

package transaction_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/mewisme/m/internal/fsx"
	"github.com/mewisme/m/internal/transaction"
)

func mklinkJunction(target, link string) error {
	out, err := exec.Command("cmd", "/c", "mklink", "/J", link, target).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, out)
	}
	return nil
}

func syscallMkfifo(path string, mode uint32) error {
	return os.ErrInvalid
}

func TestBackupTreeJunctionRoundTrip(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "store")
	if err := os.MkdirAll(filepath.Join(target, "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "vendor")
	if err := mklinkJunction(target, link); err != nil {
		t.Skip("junction not supported:", err)
	}
	if tag := fsx.ReparseTag(link); tag != fsx.IOReparseTagMountPoint {
		t.Fatalf("expected mount point tag 0x%X, got 0x%X", fsx.IOReparseTagMountPoint, tag)
	}

	txn := transaction.NewRunner(root)
	ctx := context.Background()
	if err := txn.Begin(ctx); err != nil {
		t.Fatal(err)
	}
	if err := txn.RecordBackup("vendor"); err != nil {
		t.Fatal(err)
	}

	backupPath := filepath.Join(txn.Root, "backups", "vendor.reparse.json")
	if _, err := os.Stat(backupPath); err != nil {
		t.Fatalf("expected reparse sidecar backup: %v", err)
	}
	if _, err := os.Stat(filepath.Join(txn.Root, "backups", "pkg")); os.IsNotExist(err) {
		// target not traversed into backup
	} else if err == nil {
		t.Fatal("backup must not copy junction target contents")
	}
	if _, err := os.Stat(filepath.Join(txn.Root, "backups", "vendor", "pkg")); err == nil {
		t.Fatal("backup must not traverse junction target")
	}

	if err := os.RemoveAll(link); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(link, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := txn.Rollback(ctx, transaction.StandaloneFinishOpts()); err != nil {
		t.Fatal(err)
	}
	if tag := fsx.ReparseTag(link); tag != fsx.IOReparseTagMountPoint {
		t.Fatalf("restored path is not a mount point: tag 0x%X", tag)
	}
	if _, err := os.Stat(filepath.Join(link, "pkg")); err != nil {
		t.Fatalf("junction not restored: %v", err)
	}
}
