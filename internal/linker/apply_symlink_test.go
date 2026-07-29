package linker

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestApplySymlinkSurvivesParentRename(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unprivileged directory symlinks are unavailable on Windows")
	}
	root := t.TempDir()
	stageNM := filepath.Join(root, "stage", "node_modules")
	content := filepath.Join(stageNM, ".pnpm", "pkg-a@1.0.0", "node_modules", "pkg-a")
	alias := filepath.Join(stageNM, "pkg-a")
	if err := os.MkdirAll(content, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(content, "package.json"), []byte(`{"name":"pkg-a","version":"1.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := applySymlink(content, alias); err != nil {
		t.Fatal(err)
	}
	liveNM := filepath.Join(root, "node_modules")
	if err := os.Rename(stageNM, liveNM); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(liveNM, "pkg-a", "package.json")); err != nil {
		t.Fatalf("alias broke after rename: %v", err)
	}
}
