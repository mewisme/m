package archive_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/archive"
)

func TestWriteTreePatchSingleFile(t *testing.T) {
	orig := filepath.Join(t.TempDir(), "orig")
	edit := filepath.Join(t.TempDir(), "edit")
	if err := os.MkdirAll(orig, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(edit, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(orig, "index.js"), []byte("module.exports = \"a\";\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(edit, "index.js"), []byte("module.exports = \"a-patched\";\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	patch, err := archive.WriteTreePatch(context.Background(), orig, edit)
	if err != nil {
		t.Fatal(err)
	}
	if err := archive.ApplyUnifiedPatch(context.Background(), orig, writeTempPatch(t, patch)); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(orig, "index.js"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "module.exports = \"a-patched\";\n" {
		t.Fatalf("got %q", got)
	}
}

func writeTempPatch(t *testing.T, body []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "out.patch")
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
