package archive

import (
	"runtime"
	"testing"
)

func TestResolvePatchTargetRejectsUNC(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("unc paths are windows-specific")
	}
	_, err := resolvePatchTarget("patch.patch", t.TempDir(), "\\\\server\\share\\file.txt")
	if err == nil {
		t.Fatal("expected error for unc path")
	}
}

func TestResolvePatchTargetRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	_, err := resolvePatchTarget("patch.patch", root, "../outside.txt")
	if err == nil {
		t.Fatal("expected error for traversal")
	}
}
