package transaction_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/transaction"
)

func TestGuardPathRejectsEscape(t *testing.T) {
	root := t.TempDir()
	_, err := transaction.GuardPath(root, "../outside")
	if err == nil {
		t.Fatal("expected error")
	}
	if apperr.CodeOf(err) != apperr.Transaction {
		t.Fatalf("code=%s", apperr.CodeOf(err))
	}
}

func TestGuardPathAllowsNested(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := transaction.GuardPath(root, filepath.Join("a", "b", "file.txt"))
	if err != nil {
		t.Fatal(err)
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(absRoot, "a", "b", "file.txt")
	want, err = filepath.Abs(want)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestGuardPathRejectsAbsolute(t *testing.T) {
	root := t.TempDir()
	if runtime.GOOS == "windows" {
		if _, err := transaction.GuardPath(root, `C:\outside`); err == nil {
			t.Fatal("expected error for absolute path")
		}
	} else {
		if _, err := transaction.GuardPath(root, "/etc/passwd"); err == nil {
			t.Fatal("expected error for absolute path")
		}
	}
}

func TestGuardPathRejectsMewSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	real := filepath.Join(root, "real")
	if err := os.MkdirAll(real, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(filepath.Join(root, "proj", ".mew")), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "proj", ".mew")
	if err := os.Symlink(real, link); err != nil {
		t.Skip("symlink not supported:", err)
	}
	proj := filepath.Join(root, "proj")
	_, err := transaction.GuardPath(proj, filepath.Join(".mew", "txn", "id"))
	if err == nil {
		t.Fatal("expected symlink escape rejection")
	}
}

func TestGuardPathRejectsNodeModulesSymlink(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	nm := filepath.Join(root, "node_modules", "pkg")
	if err := os.MkdirAll(filepath.Dir(nm), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, nm); err != nil {
		t.Skip("symlink not supported:", err)
	}
	_, err := transaction.GuardPath(root, filepath.Join("node_modules", "pkg", "index.js"))
	if err == nil {
		t.Fatal("expected node_modules symlink rejection")
	}
}

func TestGuardPathRejectsSnapshotsSymlink(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	snap := filepath.Join(root, ".mew", "snapshots", "s1")
	if err := os.MkdirAll(filepath.Dir(snap), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, snap); err != nil {
		t.Skip("symlink not supported:", err)
	}
	_, err := transaction.GuardPath(root, filepath.Join(".mew", "snapshots", "s1", "meta.json"))
	if err == nil {
		t.Fatal("expected snapshots symlink rejection")
	}
}

func TestRevalidatePathMatchesGuardPath(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, ".mew", "txn", "abc")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	rel := filepath.Join(".mew", "txn", "abc", "journal.json")
	got, err := transaction.RevalidatePath(root, rel)
	if err != nil {
		t.Fatal(err)
	}
	want, err := transaction.GuardPath(root, rel)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
