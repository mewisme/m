//go:build windows

package pack

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/fsx"
)

func TestPackRejectsJunctionInTree(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"p","version":"1.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "index.js"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "escape")
	if err := fsx.CreateMountPoint(link, `\\?\`+outside, outside); err != nil {
		t.Skip("junction not supported:", err)
	}
	_, err := Pack(t.Context(), Options{Root: root, PackDestination: t.TempDir()})
	if err == nil {
		t.Fatal("expected pack failure for junction")
	}
	if !strings.Contains(err.Error(), "symlink") && !strings.Contains(err.Error(), "reparse") && !strings.Contains(err.Error(), "junction") {
		t.Fatalf("unexpected error: %v", err)
	}
}
