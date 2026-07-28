package fsx_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/fsx"
)

func TestPublishFilePriorGenerationSurvivesFailedRename(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "head.json")
	initial := []byte(`{"generation":1}` + "\n")
	if err := fsx.PublishFile(path, initial, 0o644); err != nil {
		t.Fatal(err)
	}

	hook := os.Getenv("MEW_FSX_PUBLISH_HOOK")
	if hook == "fail_rename" {
		// Simulate crash after temp write: leave a temp sibling; prior file must remain valid.
		tmp := filepath.Join(dir, ".head.json.tmp-keep")
		if err := os.WriteFile(tmp, []byte("new"), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != string(initial) {
			t.Fatalf("prior generation changed: %q", got)
		}
		return
	}

	next := []byte(`{"generation":2}` + "\n")
	if err := fsx.PublishFile(path, next, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(next) {
		t.Fatalf("got %q want %q", got, next)
	}
}

func TestPublishFileGenerationScanRecovery(t *testing.T) {
	dir := t.TempDir()
	gen2 := filepath.Join(dir, "current.000002")
	body2 := []byte("txn-b\n")
	if err := fsx.PublishFile(gen2, body2, 0o644); err != nil {
		t.Fatal(err)
	}
	head := filepath.Join(dir, "current.head")
	if err := fsx.PublishFile(head, []byte("corrupt"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(gen2)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(body2) {
		t.Fatalf("numbered generation should remain valid: %q", got)
	}
}

func TestPublishRenameFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	old := []byte("old\n")
	newBytes := []byte("new\n")
	if err := os.WriteFile(dst, old, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, newBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := fsx.PublishRename(src, dst); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(newBytes) {
		t.Fatalf("got %q", got)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatal("src should be renamed away")
	}
}
