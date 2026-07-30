package integration_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func enableGlobalStore(t *testing.T) {
	t.Helper()
	t.Setenv("MEW_EXPERIMENTAL_GLOBAL_STORE", "1")
}

func TestStorePathCLI(t *testing.T) {
	projDir, cfgPath, _ := setupRegistryProject(t, `{"name":"p","version":"1.0.0"}`)
	expected := os.Getenv("MEW_STORE_DIR")
	code, out := runM(t, projDir, cfgPath, "store", "path")
	if code != 0 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	got := strings.TrimSpace(out)
	if got != expected {
		t.Fatalf("path=%q want %q", got, expected)
	}
}

func TestStoreDedupAcrossInstalls(t *testing.T) {
	enableGlobalStore(t)
	pkgJSON := `{"name":"dedup-a","version":"1.0.0","dependencies":{"lodash":"4.17.21"}}`
	projA, cfgA, _ := setupRegistryProject(t, pkgJSON)
	projB := t.TempDir()
	if err := os.WriteFile(filepath.Join(projB, "package.json"), []byte(`{"name":"dedup-b","version":"1.0.0","dependencies":{"lodash":"4.17.21"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfgB := filepath.Join(projB, "m.jsonc")
	cfgData, err := os.ReadFile(cfgA)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfgB, cfgData, 0o644); err != nil {
		t.Fatal(err)
	}
	if code, out := runM(t, projA, cfgA, "install"); code != 0 {
		t.Fatalf("install A: %s", out)
	}
	if code, out := runM(t, projB, cfgB, "install"); code != 0 {
		t.Fatalf("install B: %s", out)
	}
	storeRoot := os.Getenv("MEW_STORE_DIR")
	entries, err := os.ReadDir(filepath.Join(storeRoot, "packages", "sha256"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one lodash store entry, got %d", len(entries))
	}
}

func TestStorePruneDryRun(t *testing.T) {
	enableGlobalStore(t)
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "prune-test",
  "version": "1.0.0",
  "dependencies": { "lodash": "4.17.21" }
}`)
	if code, out := runM(t, projDir, cfgPath, "install"); code != 0 {
		t.Fatalf("install: %s", out)
	}
	code, out := runM(t, projDir, cfgPath, "store", "prune", "--dry-run")
	if code != 0 {
		t.Fatalf("prune: %s", out)
	}
	if !strings.Contains(out, "removed") || !strings.Contains(out, "0") {
		t.Fatalf("expected referenced package kept: %s", out)
	}
}

func TestSmartLinkInstall(t *testing.T) {
	enableGlobalStore(t)
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "smart-link",
  "version": "1.0.0",
  "dependencies": { "lodash": "4.17.21" }
}`)
	code, out := runM(t, projDir, cfgPath, "install")
	if code != 0 {
		t.Fatalf("install: %s", out)
	}
	if _, err := os.Stat(filepath.Join(projDir, "node_modules", "lodash", "package.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(projDir, ".mew", "store-manifest.json")); err != nil {
		t.Fatal("store manifest missing")
	}
}

func TestStoreMutationIsolation(t *testing.T) {
	enableGlobalStore(t)
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "mutation",
  "version": "1.0.0",
  "dependencies": { "lodash": "4.17.21" }
}`)
	if code, out := runM(t, projDir, cfgPath, "install"); code != 0 {
		t.Fatalf("install: %s", out)
	}
	nmFile := filepath.Join(projDir, "node_modules", "lodash", "package.json")
	before, err := os.ReadFile(nmFile)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(nmFile, []byte(`{"name":"tampered"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	storeRoot := os.Getenv("MEW_STORE_DIR")
	entries, err := os.ReadDir(filepath.Join(storeRoot, "packages", "sha256"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one store entry, got %d", len(entries))
	}
	storePkg := filepath.Join(storeRoot, "packages", "sha256", entries[0].Name(), "package.json")
	storeData, err := os.ReadFile(storePkg)
	if err != nil {
		t.Fatal(err)
	}
	if string(storeData) == `{"name":"tampered"}` {
		t.Fatal("store package was mutated via node_modules link")
	}
	if string(storeData) != string(before) {
		t.Fatal("store package content changed after node_modules mutation")
	}
	if _, err := os.Stat(filepath.Join(storeRoot, "packages", "sha256", entries[0].Name(), ".mew-tree-manifest.json")); err != nil {
		t.Fatal("tree manifest missing in store")
	}
}
