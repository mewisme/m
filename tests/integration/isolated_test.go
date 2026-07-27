package integration_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/m/internal/testkit"
)

func enableIsolatedLinker(t *testing.T) {
	t.Helper()
	t.Setenv("MEW_EXPERIMENTAL_ISOLATED_LINKER", "1")
}

func TestIsolatedInstallBasicLayout(t *testing.T) {
	enableIsolatedLinker(t)
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "isolated-basic",
  "version": "1.0.0",
  "dependencies": { "pkg-a": "^1.0.0" }
}`)
	code, out := runM(t, projDir, cfgPath, "install", "--linker=isolated")
	if code != 0 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	pnpmDir := filepath.Join(projDir, "node_modules", ".pnpm")
	if _, err := os.Stat(pnpmDir); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(projDir, "node_modules", "pkg-a", "package.json")
	if _, err := os.Stat(alias); err != nil {
		t.Fatal(err)
	}
	modulesMeta := filepath.Join(projDir, "node_modules", ".mew", "modules.v1.json")
	if _, err := os.Stat(modulesMeta); err != nil {
		t.Fatal(err)
	}
}

func TestIsolatedLockPersistsLinkerMode(t *testing.T) {
	enableIsolatedLinker(t)
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "isolated-lock",
  "version": "1.0.0",
  "dependencies": { "lodash": "4.17.21" }
}`)
	code, out := runM(t, projDir, cfgPath, "install", "--linker=isolated")
	if code != 0 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	raw, err := os.ReadFile(filepath.Join(projDir, "m.lock"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"linker": "isolated"`) {
		t.Fatalf("m.lock missing isolated linker:\n%s", raw)
	}
}

func TestIsolatedPhantomDependencyBlocked(t *testing.T) {
	enableIsolatedLinker(t)
	projDir := t.TempDir()
	testkit.CopyFixture(t, "projects/phantom-dep-negative", projDir)
	reg := testkit.LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)
	cfgPath := filepath.Join(projDir, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte(`{"registry":"`+srv.URL+`","install.linker":"isolated"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out := runM(t, projDir, cfgPath, "install")
	if code != 0 {
		t.Fatalf("install exit=%d out=%s", code, out)
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not in PATH")
	}
	cmd := exec.Command(node, "index.js")
	cmd.Dir = projDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("phantom dep reachable: %v", err)
	}
	cmd = exec.Command(node, "-e", "require('pkg-b')")
	cmd.Dir = projDir
	if out, err := cmd.CombinedOutput(); err == nil {
		t.Fatalf("phantom require succeeded: %s", out)
	}
	cmd = exec.Command(node, "-e", "require('pkg-a'); console.log('ok')")
	cmd.Dir = projDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("declared dep should resolve: %v\n%s", err, out)
	}
}

func TestIsolatedBinShimsCreated(t *testing.T) {
	enableIsolatedLinker(t)
	projDir := t.TempDir()
	testkit.CopyFixture(t, "projects/bin-shims", projDir)
	reg := testkit.LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)
	cfgPath := filepath.Join(projDir, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte(`{"registry":"`+srv.URL+`"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out := runM(t, projDir, cfgPath, "install", "--linker=isolated")
	if code != 0 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	binDir := filepath.Join(projDir, "node_modules", ".bin")
	if _, err := os.Stat(filepath.Join(binDir, "cli")); err == nil {
		return
	}
	if _, err := os.Stat(filepath.Join(binDir, "cli.cmd")); err == nil {
		return
	}
	t.Fatalf("no cli shim in %s", binDir)
}

func TestIsolatedScopedPackage(t *testing.T) {
	enableIsolatedLinker(t)
	projDir := t.TempDir()
	testkit.CopyFixture(t, "projects/isolated-peers", projDir)
	reg := testkit.LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)
	cfgPath := filepath.Join(projDir, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte(`{"registry":"`+srv.URL+`"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out := runM(t, projDir, cfgPath, "install", "--linker=isolated")
	if code != 0 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	alias := filepath.Join(projDir, "node_modules", "@scope", "pkg", "package.json")
	if _, err := os.Stat(alias); err != nil {
		t.Fatal(err)
	}
}

func TestHoistedLinkerFlagStillWorks(t *testing.T) {
	enableIsolatedLinker(t)
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "hoisted-explicit",
  "version": "1.0.0",
  "dependencies": { "lodash": "4.17.21" }
}`)
	code, out := runM(t, projDir, cfgPath, "install", "--linker=hoisted")
	if code != 0 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	if _, err := os.Stat(filepath.Join(projDir, "node_modules", ".pnpm")); err == nil {
		t.Fatal("hoisted install should not create .pnpm")
	}
}

func TestIsolatedRequiresExperimentalGate(t *testing.T) {
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "gate",
  "version": "1.0.0",
  "dependencies": { "lodash": "4.17.21" }
}`)
	code, _ := runM(t, projDir, cfgPath, "install", "--linker=isolated")
	if code == 0 {
		t.Fatal("expected failure without experimental gate")
	}
}

func TestModulesMetadataShape(t *testing.T) {
	enableIsolatedLinker(t)
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "meta",
  "version": "1.0.0",
  "dependencies": { "pkg-a": "^1.0.0" }
}`)
	code, out := runM(t, projDir, cfgPath, "install", "--linker=isolated")
	if code != 0 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	raw, err := os.ReadFile(filepath.Join(projDir, "node_modules", ".mew", "modules.v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Linker   string `json:"linker"`
		Packages []struct {
			Key     string `json:"key"`
			StoreID string `json:"storeID"`
		} `json:"packages"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Linker != "isolated" || len(doc.Packages) == 0 {
		t.Fatalf("bad modules doc: %+v", doc)
	}
}
