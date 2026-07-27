//go:build windows

package transaction_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func assertJunctionTarget(t *testing.T, link, wantTarget string) {
	t.Helper()
	if tag := fsx.ReparseTag(link); tag != fsx.IOReparseTagMountPoint {
		t.Fatalf("%s: expected mount point tag 0x%X, got 0x%X", link, fsx.IOReparseTagMountPoint, tag)
	}
	if _, err := os.Stat(filepath.Join(link, wantTarget)); err != nil {
		t.Fatalf("%s: junction target %q not reachable: %v", link, wantTarget, err)
	}
}

func beginTxn(t *testing.T, root string) (*transaction.Runner, context.Context) {
	t.Helper()
	txn := transaction.NewRunner(root)
	ctx := context.Background()
	if err := txn.Begin(ctx); err != nil {
		t.Fatal(err)
	}
	return txn, ctx
}

func rollbackTxn(t *testing.T, txn *transaction.Runner, ctx context.Context) {
	t.Helper()
	if _, err := txn.Rollback(ctx, transaction.StandaloneFinishOpts()); err != nil {
		t.Fatal(err)
	}
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

	txn, ctx := beginTxn(t, root)
	if err := txn.RecordBackup("vendor"); err != nil {
		t.Fatal(err)
	}

	metaPath := filepath.Join(txn.Root, "backups-meta", "vendor.json")
	if _, err := os.Stat(metaPath); err != nil {
		t.Fatalf("expected junction metadata at %s: %v", metaPath, err)
	}
	if _, err := os.Stat(filepath.Join(txn.Root, "backups", "vendor.reparse.json")); !os.IsNotExist(err) {
		t.Fatal("legacy in-tree reparse sidecar must not exist")
	}
	if _, err := os.Stat(filepath.Join(txn.Root, "backups", "vendor")); !os.IsNotExist(err) {
		t.Fatal("junction must not be mirrored in backups tree")
	}

	if err := os.RemoveAll(link); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(link, 0o755); err != nil {
		t.Fatal(err)
	}
	rollbackTxn(t, txn, ctx)
	assertJunctionTarget(t, link, "pkg")
}

func TestBackupTreeNestedJunctionRoundTrip(t *testing.T) {
	root := t.TempDir()
	store := filepath.Join(root, "store", "package-b")
	if err := os.MkdirAll(filepath.Join(store, "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	nm := filepath.Join(root, "node_modules")
	if err := os.MkdirAll(nm, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(nm, "package-b")
	if err := mklinkJunction(store, link); err != nil {
		t.Skip("junction not supported:", err)
	}

	txn, ctx := beginTxn(t, root)
	if err := txn.RecordBackup("node_modules"); err != nil {
		t.Fatal(err)
	}

	metaPath := filepath.Join(txn.Root, "backups-meta", "node_modules", "package-b.json")
	if _, err := os.Stat(metaPath); err != nil {
		t.Fatalf("expected nested junction metadata: %v", err)
	}

	if err := os.RemoveAll(nm); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(nm, 0o755); err != nil {
		t.Fatal(err)
	}
	rollbackTxn(t, txn, ctx)
	assertJunctionTarget(t, link, "lib")
}

func TestBackupTreeScopedJunctionRoundTrip(t *testing.T) {
	root := t.TempDir()
	store := filepath.Join(root, "store", "scoped")
	if err := os.MkdirAll(filepath.Join(store, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	scopeDir := filepath.Join(root, "node_modules", "@scope")
	if err := os.MkdirAll(scopeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(scopeDir, "pkg")
	if err := mklinkJunction(store, link); err != nil {
		t.Skip("junction not supported:", err)
	}

	txn, ctx := beginTxn(t, root)
	if err := txn.RecordBackup("node_modules"); err != nil {
		t.Fatal(err)
	}

	metaPath := filepath.Join(txn.Root, "backups-meta", "node_modules", "@scope", "pkg.json")
	if _, err := os.Stat(metaPath); err != nil {
		t.Fatalf("expected scoped junction metadata: %v", err)
	}

	if err := os.RemoveAll(filepath.Join(root, "node_modules")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "node_modules"), 0o755); err != nil {
		t.Fatal(err)
	}
	rollbackTxn(t, txn, ctx)
	assertJunctionTarget(t, link, "bin")
}

func TestBackupTreeIsolatedLayoutJunctionRoundTrip(t *testing.T) {
	root := t.TempDir()
	store := filepath.Join(root, "store", "pkg")
	if err := os.MkdirAll(filepath.Join(store, "dist"), 0o755); err != nil {
		t.Fatal(err)
	}
	iso := filepath.Join(root, "node_modules", ".pnpm", "pkg@1.0.0", "node_modules", "pkg")
	if err := os.MkdirAll(filepath.Dir(iso), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := mklinkJunction(store, iso); err != nil {
		t.Skip("junction not supported:", err)
	}

	txn, ctx := beginTxn(t, root)
	if err := txn.RecordBackup("node_modules"); err != nil {
		t.Fatal(err)
	}

	metaRel := filepath.Join("backups-meta", "node_modules", ".pnpm", "pkg@1.0.0", "node_modules", "pkg.json")
	if _, err := os.Stat(filepath.Join(txn.Root, metaRel)); err != nil {
		t.Fatalf("expected isolated-layout junction metadata: %v", err)
	}

	if err := os.RemoveAll(filepath.Join(root, "node_modules")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "node_modules"), 0o755); err != nil {
		t.Fatal(err)
	}
	rollbackTxn(t, txn, ctx)
	assertJunctionTarget(t, iso, "dist")
}

func TestBackupTreeReparseJSONFileRoundTrip(t *testing.T) {
	root := t.TempDir()
	nm := filepath.Join(root, "node_modules")
	if err := os.MkdirAll(nm, 0o755); err != nil {
		t.Fatal(err)
	}
	payload := []byte(`{"note":"user file"}`)
	filePath := filepath.Join(nm, "foo.reparse.json")
	if err := os.WriteFile(filePath, payload, 0o644); err != nil {
		t.Fatal(err)
	}

	txn, ctx := beginTxn(t, root)
	if err := txn.RecordBackup("node_modules"); err != nil {
		t.Fatal(err)
	}

	backupFile := filepath.Join(txn.Root, "backups", "node_modules", "foo.reparse.json")
	got, err := os.ReadFile(backupFile)
	if err != nil {
		t.Fatalf("expected mirrored regular file backup: %v", err)
	}
	if string(got) != string(payload) {
		t.Fatalf("backup payload=%q want %q", got, payload)
	}
	metaPath := filepath.Join(txn.Root, "backups-meta", "node_modules", "foo.reparse.json.json")
	if _, err := os.Stat(metaPath); !os.IsNotExist(err) {
		t.Fatal("regular foo.reparse.json must not produce junction metadata")
	}

	if err := os.WriteFile(filePath, []byte("mutated"), 0o644); err != nil {
		t.Fatal(err)
	}
	rollbackTxn(t, txn, ctx)
	restored, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(restored) != string(payload) {
		t.Fatalf("restored payload=%q want %q", restored, payload)
	}
	if fsx.ReparseTag(filePath) != 0 {
		t.Fatal("foo.reparse.json must remain a regular file, not a junction")
	}
}

func TestRestoreJunctionMetaRejectsMalformed(t *testing.T) {
	root := t.TempDir()
	nm := filepath.Join(root, "node_modules")
	if err := os.MkdirAll(nm, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nm, "keep.txt"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}

	txn, ctx := beginTxn(t, root)
	if err := txn.RecordBackup("node_modules"); err != nil {
		t.Fatal(err)
	}

	metaDir := filepath.Join(txn.Root, "backups-meta", "node_modules")
	if err := os.MkdirAll(metaDir, 0o755); err != nil {
		t.Fatal(err)
	}
	badMeta := filepath.Join(metaDir, "evil.json")
	if err := os.WriteFile(badMeta, []byte(`{"schemaVersion":1,"relPath":"../outside","tag":2684354563,"substitute":"\\??\\C:\\","print":"C:\\","entryType":"junction"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.RemoveAll(nm); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(nm, 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := txn.Rollback(ctx, transaction.StandaloneFinishOpts())
	if err == nil {
		t.Fatal("expected malformed junction metadata to fail restore")
	}
	if !strings.Contains(err.Error(), "relPath") && !strings.Contains(err.Error(), "escapes") {
		t.Fatalf("expected traversal rejection, got: %v", err)
	}
}

func TestRestoreJunctionMetaRejectsUnsupportedTag(t *testing.T) {
	root := t.TempDir()
	nm := filepath.Join(root, "node_modules")
	if err := os.MkdirAll(nm, 0o755); err != nil {
		t.Fatal(err)
	}

	txn, ctx := beginTxn(t, root)
	if err := txn.RecordBackup("node_modules"); err != nil {
		t.Fatal(err)
	}

	meta := map[string]any{
		"schemaVersion": 1,
		"relPath":       "node_modules/pkg",
		"tag":           0xA000000C,
		"substitute":    `\??\C:\store`,
		"print":         `C:\store`,
		"entryType":     "junction",
	}
	data, err := json.Marshal(meta)
	if err != nil {
		t.Fatal(err)
	}
	metaPath := filepath.Join(txn.Root, "backups-meta", "node_modules", "pkg.json")
	if err := os.MkdirAll(filepath.Dir(metaPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(metaPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.RemoveAll(nm); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(nm, 0o755); err != nil {
		t.Fatal(err)
	}
	_, err = txn.Rollback(ctx, transaction.StandaloneFinishOpts())
	if err == nil {
		t.Fatal("expected unsupported tag to fail restore")
	}
	if !strings.Contains(err.Error(), "unsupported reparse tag") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBackupTreeNodeModulesMultiJunctionRollback(t *testing.T) {
	root := t.TempDir()
	storeA := filepath.Join(root, "store-a")
	storeB := filepath.Join(root, "store-b")
	if err := os.MkdirAll(filepath.Join(storeA, "a"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(storeB, "b"), 0o755); err != nil {
		t.Fatal(err)
	}
	nm := filepath.Join(root, "node_modules")
	if err := os.MkdirAll(nm, 0o755); err != nil {
		t.Fatal(err)
	}
	linkA := filepath.Join(nm, "package-a")
	linkB := filepath.Join(nm, "package-b")
	if err := mklinkJunction(storeA, linkA); err != nil {
		t.Skip("junction not supported:", err)
	}
	if err := mklinkJunction(storeB, linkB); err != nil {
		t.Skip("junction not supported:", err)
	}

	txn, ctx := beginTxn(t, root)
	if err := txn.RecordBackup("node_modules"); err != nil {
		t.Fatal(err)
	}

	for _, rel := range []string{"node_modules/package-a.json", "node_modules/package-b.json"} {
		if _, err := os.Stat(filepath.Join(txn.Root, "backups-meta", rel)); err != nil {
			t.Fatalf("expected metadata for %s: %v", rel, err)
		}
	}

	if err := os.RemoveAll(nm); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(nm, 0o755); err != nil {
		t.Fatal(err)
	}
	rollbackTxn(t, txn, ctx)
	assertJunctionTarget(t, linkA, "a")
	assertJunctionTarget(t, linkB, "b")
}
