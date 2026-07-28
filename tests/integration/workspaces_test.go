package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/m/internal/lockfile/mlock"
	"github.com/mewisme/m/internal/testkit"
)

func setupWorkspaceProject(t *testing.T, fixtureRel string) (projDir, cfgPath string) {
	t.Helper()
	testkit.EnableWorkspaces(t)
	testkit.CleanEnv(t)
	t.Setenv("NO_PROXY", "*")
	t.Setenv("no_proxy", "*")

	reg := testkit.LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)

	projDir = t.TempDir()
	testkit.CopyFixture(t, fixtureRel, projDir)
	cfgPath = filepath.Join(projDir, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte(`{"registry":"`+srv.URL+`"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return projDir, cfgPath
}

func readLockImporters(t *testing.T, projDir string) map[string]mlock.ImporterSection {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(projDir, "m.lock"))
	if err != nil {
		t.Fatal(err)
	}
	var doc mlock.Document
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	out := make(map[string]mlock.ImporterSection, len(doc.Importers))
	for _, im := range doc.Importers {
		out[string(im.ID)] = im
	}
	return out
}

func TestWorkspaceInstallRecursive(t *testing.T) {
	projDir, cfgPath := setupWorkspaceProject(t, "projects/workspace-simple")
	code, out := runM(t, projDir, cfgPath, "install", "-r")
	if code != 0 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	importers := readLockImporters(t, projDir)
	if _, ok := importers["packages/a"]; !ok {
		t.Fatalf("missing packages/a importer: %v", importers)
	}
	if _, err := os.Stat(filepath.Join(projDir, "node_modules", ".pnpm")); err != nil {
		t.Fatal(err)
	}
	lockData, err := os.ReadFile(filepath.Join(projDir, "m.lock"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(lockData), "lodash@4.17.21") {
		t.Fatalf("lock missing lodash: %s", string(lockData))
	}
}

func TestWorkspaceCatalogInstall(t *testing.T) {
	projDir, cfgPath := setupWorkspaceProject(t, "projects/workspace-catalog")
	code, out := runM(t, projDir, cfgPath, "install", "-r")
	if code != 0 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	data, err := os.ReadFile(filepath.Join(projDir, "m.lock"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "lodash@4.17.21") {
		t.Fatalf("lock missing catalog-resolved lodash: %s", string(data))
	}
}

func TestWorkspaceFilterPreservesLock(t *testing.T) {
	projDir, cfgPath := setupWorkspaceProject(t, "projects/workspace-filter")
	code, out := runM(t, projDir, cfgPath, "install", "-r")
	if code != 0 {
		t.Fatalf("full install exit=%d out=%s", code, out)
	}
	before := readLockImporters(t, projDir)
	code, out = runM(t, projDir, cfgPath, "--filter", "alpha", "install")
	if code != 0 {
		t.Fatalf("filter install exit=%d out=%s", code, out)
	}
	after := readLockImporters(t, projDir)
	if _, ok := after["packages/beta"]; !ok {
		t.Fatal("beta importer section dropped after filter install")
	}
	if len(before["packages/beta"].Specifiers) != len(after["packages/beta"].Specifiers) {
		t.Fatalf("beta specifiers changed: before=%d after=%d", len(before["packages/beta"].Specifiers), len(after["packages/beta"].Specifiers))
	}
	lockData, err := os.ReadFile(filepath.Join(projDir, "m.lock"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(lockData), "pkg-b@1.2.0") {
		t.Fatalf("pkg-b missing from lock after filtered install: %s", string(lockData))
	}
	nm := filepath.Join(projDir, "node_modules")
	found := false
	_ = filepath.Walk(nm, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.Name() == "package.json" && filepath.Base(filepath.Dir(path)) == "pkg-b" {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	if !found {
		t.Fatal("pkg-b not linkable under node_modules after filtered install")
	}
}

func TestWorkspaceAddFilterUpdatesMemberOnly(t *testing.T) {
	projDir, cfgPath := setupWorkspaceProject(t, "projects/workspace-filter")
	code, out := runM(t, projDir, cfgPath, "install", "-r")
	if code != 0 {
		t.Fatalf("install exit=%d out=%s", code, out)
	}
	rootBefore, _ := os.ReadFile(filepath.Join(projDir, "package.json"))
	betaBefore, _ := os.ReadFile(filepath.Join(projDir, "packages", "beta", "package.json"))
	code, out = runM(t, projDir, cfgPath, "--filter", "alpha", "add", "lodash")
	if code != 0 {
		t.Fatalf("add exit=%d out=%s", code, out)
	}
	rootAfter, _ := os.ReadFile(filepath.Join(projDir, "package.json"))
	if string(rootBefore) != string(rootAfter) {
		t.Fatal("root package.json changed on filtered add")
	}
	alphaData, err := os.ReadFile(filepath.Join(projDir, "packages", "alpha", "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(alphaData), "lodash") {
		t.Fatalf("alpha missing lodash: %s", string(alphaData))
	}
	betaAfter, _ := os.ReadFile(filepath.Join(projDir, "packages", "beta", "package.json"))
	if string(betaBefore) != string(betaAfter) {
		t.Fatal("beta package.json changed on filtered add to alpha")
	}
}

func TestCiRejectsFilter(t *testing.T) {
	projDir, cfgPath := setupWorkspaceProject(t, "projects/workspace-simple")
	code, out := runM(t, projDir, cfgPath, "install", "-r")
	if code != 0 {
		t.Fatalf("install exit=%d out=%s", code, out)
	}
	code, out = runM(t, projDir, cfgPath, "--filter", "a", "ci")
	if code == 0 {
		t.Fatalf("expected ci --filter to fail, out=%s", out)
	}
	_ = out
}

func TestWorkspaceProtocolMissing(t *testing.T) {
	projDir, cfgPath := setupWorkspaceProject(t, "projects/workspace-protocol")
	pkgPath := filepath.Join(projDir, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(string(data), `"lib": "workspace:*"`, `"missing-pkg": "workspace:*"`, 1)
	if err := os.WriteFile(pkgPath, []byte(updated), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out := runM(t, projDir, cfgPath, "install")
	if code == 0 {
		t.Fatalf("expected resolve failure, out=%s", out)
	}
}
