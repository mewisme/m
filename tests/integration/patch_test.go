package integration_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/testkit"
)

func TestPatchDeterministicAcrossInstalls(t *testing.T) {
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "patch-left-pad",
  "version": "1.0.0",
  "private": true,
  "dependencies": { "pkg-a": "1.0.0" }
}`)
	code, out := runM(t, projDir, cfgPath, "install")
	if code != 0 {
		t.Fatalf("install exit=%d out=%s", code, out)
	}

	code, out = runM(t, projDir, cfgPath, "patch", "pkg-a")
	if code != 0 {
		t.Fatalf("patch extract exit=%d out=%s", code, out)
	}
	// Status rendering prepends a symbol (e.g. "• ") — strip it.
	editDir := strings.TrimSpace(out)
	if idx := strings.Index(editDir, " "); idx >= 0 {
		editDir = strings.TrimSpace(editDir[idx+1:])
	}
	indexPath := filepath.Join(editDir, "index.js")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	patched := strings.Replace(string(data), `"pkg-a"`, `"pkg-a-patched"`, 1)
	if err := os.WriteFile(indexPath, []byte(patched), 0o644); err != nil {
		t.Fatal(err)
	}

	code, out = runM(t, projDir, cfgPath, "patch", "pkg-a", "--commit")
	if code != 0 {
		t.Fatalf("patch commit exit=%d out=%s", code, out)
	}
	patchFile := filepath.Join(projDir, "patches", "pkg-a@1.0.0.patch")
	if _, err := os.Stat(patchFile); err != nil {
		t.Fatal(err)
	}
	pkgJSON, err := os.ReadFile(filepath.Join(projDir, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(pkgJSON), `"patchedDependencies"`) || !strings.Contains(string(pkgJSON), "patches/pkg-a@1.0.0.patch") {
		t.Fatalf("package.json missing patchedDependencies: %s", pkgJSON)
	}

	first := readInstalledPkgA(t, projDir)
	code, out = runM(t, projDir, cfgPath, "install")
	if code != 0 {
		t.Fatalf("second install exit=%d out=%s", code, out)
	}
	second := readInstalledPkgA(t, projDir)
	if first != second {
		t.Fatalf("patched content not deterministic:\nfirst=%q\nsecond=%q", first, second)
	}
	if !strings.Contains(first, "pkg-a-patched") {
		t.Fatalf("expected patched marker, got %q", first)
	}
}

func readInstalledPkgA(t *testing.T, projDir string) string {
	t.Helper()
	path := filepath.Join(projDir, "node_modules", "pkg-a", "index.js")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestPatchFixtureSourceTree(t *testing.T) {
	dir := testkit.FixtureDir(t, "sources/patch-left-pad")
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "pkg-a") {
		t.Fatalf("fixture package.json unexpected: %s", data)
	}
}
