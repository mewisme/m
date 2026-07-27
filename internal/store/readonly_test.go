package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMakeTreeReadOnlyRoundTrip(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "pkg", "index.js")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := makeTreeReadOnly(root); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o444 {
		t.Fatalf("mode=%o want 0444", info.Mode().Perm())
	}
	t.Cleanup(func() { _ = makeTreeWritable(root) })
}
