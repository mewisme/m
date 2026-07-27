package fsx_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/fsx"
)

func TestReplaceExistingFilePreservesDestDuringFailure(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "dst.txt")
	old := []byte("old-content\n")
	if err := os.WriteFile(dst, old, 0o644); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(dir, "missing-src.txt")
	err := fsx.ReplaceExistingFile(missing, dst)
	if err == nil {
		t.Fatal("expected error from missing src")
	}
	got, readErr := os.ReadFile(dst)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(got) != string(old) {
		t.Fatalf("dest corrupted during failed replace: %q", got)
	}
}

func TestReplaceExistingFileReplacesDest(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	if err := os.WriteFile(dst, []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	newBytes := []byte("new\n")
	if err := os.WriteFile(src, newBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := fsx.ReplaceExistingFile(src, dst); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(newBytes) {
		t.Fatalf("got %q", got)
	}
}

func TestWriteGenerationExclusiveRejectsDuplicate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "journal.000001.json")
	body := []byte(`{"generation":1}` + "\n")
	if err := fsx.WriteGenerationExclusive(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
	err := fsx.WriteGenerationExclusive(path, body, 0o644)
	if err == nil {
		t.Fatal("expected duplicate generation error")
	}
	if apperr.CodeOf(err) != apperr.Integrity {
		t.Fatalf("code=%s", apperr.CodeOf(err))
	}
}

func TestPublishDirectorySyncsParent(t *testing.T) {
	root := t.TempDir()
	live := filepath.Join(root, "node_modules")
	stage := filepath.Join(root, "stage_nm")
	if err := os.MkdirAll(filepath.Join(live, "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(live, "pkg", "index.js"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(stage, "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stage, "pkg", "index.js"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := fsx.PublishDirectory(stage, live); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(live, "pkg", "index.js"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new" {
		t.Fatalf("got %q", got)
	}
}
