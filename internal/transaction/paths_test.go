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
	absRoot, err = filepath.EvalSymlinks(absRoot)
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
