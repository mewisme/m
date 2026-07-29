package integration_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/lockfile"
	"github.com/mewisme/mew/internal/lockfile/mlock"
	"github.com/mewisme/mew/internal/project"
	"github.com/mewisme/mew/internal/testkit"
	"github.com/mewisme/mew/internal/transaction"
)

func setupDedupeFixture(t *testing.T) (projDir, cfgPath string) {
	t.Helper()
	testkit.CleanEnv(t)
	t.Setenv("NO_PROXY", "*")
	reg := testkit.LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)
	projDir = t.TempDir()
	testkit.CopyFixture(t, "projects/dedupe-duplicates", projDir)
	cfgPath = filepath.Join(projDir, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte(`{"registry":"`+srv.URL+`"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out := runM(t, projDir, cfgPath, "install")
	if code != 0 {
		t.Fatalf("install exit=%d out=%s", code, out)
	}
	injectDuplicatePkgB(t, projDir)
	return projDir, cfgPath
}

func injectDuplicatePkgB(t *testing.T, projDir string) {
	t.Helper()
	g, err := lockfile.ReadGraph(context.Background(), projDir, project.IdentityMew)
	if err != nil {
		t.Fatal(err)
	}
	var pkgB100 graph.Package
	for _, p := range g.Packages {
		if p.ID.Name == "pkg-b" && p.ID.Version == "1.0.0" {
			pkgB100 = p
			break
		}
	}
	if pkgB100.ID.Name == "" {
		for _, p := range g.Packages {
			if p.ID.Name == "pkg-b" {
				pkgB100 = p
				pkgB100.ID.Version = "1.0.0"
				pkgB100.Integrity = "sha256-bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
				pkgB100.TarballURL = "pkg-b-1.0.0.tgz"
				break
			}
		}
	}
	if pkgB100.ID.Name == "" {
		t.Fatal("pkg-b missing from lock graph")
	}
	b := graph.NewBuilder()
	for _, im := range g.Importers {
		b.Importer(im.ID, im.Name)
	}
	seen := map[string]struct{}{}
	for _, p := range g.Packages {
		b.Package(p.ID, p.Integrity, p.TarballURL)
		seen[p.ID.Key()] = struct{}{}
	}
	dupKey := graph.PackageID{Name: "pkg-b", Version: "1.0.0"}.Key()
	if _, ok := seen[dupKey]; !ok {
		b.Package(pkgB100.ID, pkgB100.Integrity, pkgB100.TarballURL)
	}
	for _, e := range g.Edges {
		b.EdgeEx(e.From, e.Name, e.To, e.Kind, e.Range, e.Optional)
	}
	dupGraph, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}
	if err := (mlock.Adapter{}).Write(context.Background(), filepath.Join(projDir, "m.lock"), dupGraph); err != nil {
		t.Fatal(err)
	}
}

func lockPackageCount(t *testing.T, projDir string) int {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(projDir, "m.lock"))
	if err != nil {
		t.Fatal(err)
	}
	var doc mlock.Document
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	return len(doc.Packages)
}

func TestDedupeReducesLockDuplicates(t *testing.T) {
	projDir, cfgPath := setupDedupeFixture(t)
	before := lockPackageCount(t, projDir)
	if before < 3 {
		t.Fatalf("expected duplicate lock entries, got %d packages", before)
	}
	code, out := runM(t, projDir, cfgPath, "dedupe")
	if code != 0 {
		t.Fatalf("dedupe exit=%d out=%s", code, out)
	}
	after := lockPackageCount(t, projDir)
	if after >= before {
		t.Fatalf("expected fewer packages after dedupe: before=%d after=%d", before, after)
	}
}

func TestDedupeDryRunPreservesLock(t *testing.T) {
	projDir, cfgPath := setupDedupeFixture(t)
	lockBefore, err := os.ReadFile(filepath.Join(projDir, "m.lock"))
	if err != nil {
		t.Fatal(err)
	}
	code, out := runM(t, projDir, cfgPath, "dedupe", "--dry-run")
	if code != 0 {
		t.Fatalf("dedupe dry-run exit=%d out=%s", code, out)
	}
	lockAfter, err := os.ReadFile(filepath.Join(projDir, "m.lock"))
	if err != nil {
		t.Fatal(err)
	}
	if string(lockAfter) != string(lockBefore) {
		t.Fatal("lock bytes changed after dedupe --dry-run")
	}
}

func TestDedupeTxnInjectRollback(t *testing.T) {
	projDir, cfgPath := setupDedupeFixture(t)
	lockBefore, err := os.ReadFile(filepath.Join(projDir, "m.lock"))
	if err != nil {
		t.Fatal(err)
	}
	transaction.SetTestHook(func(phase string, opIndex int) error {
		if phase == "publish" && opIndex == 0 {
			return os.ErrPermission
		}
		return nil
	})
	t.Cleanup(func() { transaction.SetTestHook(nil) })

	code, out := runM(t, projDir, cfgPath, "dedupe")
	if code == 0 {
		t.Fatalf("expected dedupe failure, out=%s", out)
	}
	lockAfter, err := os.ReadFile(filepath.Join(projDir, "m.lock"))
	if err != nil {
		t.Fatal(err)
	}
	if string(lockAfter) != string(lockBefore) {
		t.Fatal("lock changed after failed dedupe commit")
	}
	_, _ = runM(t, projDir, cfgPath, "recover")
}

func TestPruneRemovesExtraneous(t *testing.T) {
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "prune-test",
  "version": "1.0.0",
  "dependencies": { "lodash": "4.17.21" }
}`)
	code, out := runM(t, projDir, cfgPath, "install")
	if code != 0 {
		t.Fatalf("install exit=%d out=%s", code, out)
	}
	leftover := filepath.Join(projDir, "node_modules", "leftover-pkg")
	if err := os.MkdirAll(leftover, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(leftover, "package.json"), []byte(`{"name":"leftover-pkg","version":"1.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out = runM(t, projDir, cfgPath, "prune")
	if code != 0 {
		t.Fatalf("prune exit=%d out=%s", code, out)
	}
	if _, err := os.Stat(leftover); !os.IsNotExist(err) {
		t.Fatal("leftover-pkg should be removed")
	}
	if _, err := os.Stat(filepath.Join(projDir, "node_modules", "lodash", "package.json")); err != nil {
		t.Fatal("lodash should remain")
	}
}

func TestPruneDryRunKeepsExtraneous(t *testing.T) {
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "prune-dry",
  "version": "1.0.0",
  "dependencies": { "lodash": "4.17.21" }
}`)
	if code, out := runM(t, projDir, cfgPath, "install"); code != 0 {
		t.Fatalf("install exit=%d out=%s", code, out)
	}
	leftover := filepath.Join(projDir, "node_modules", "leftover-pkg")
	if err := os.MkdirAll(leftover, 0o755); err != nil {
		t.Fatal(err)
	}
	code, out := runM(t, projDir, cfgPath, "prune", "--dry-run")
	if code != 0 {
		t.Fatalf("prune dry-run exit=%d out=%s", code, out)
	}
	if _, err := os.Stat(leftover); err != nil {
		t.Fatal("leftover-pkg should remain after dry-run")
	}
}

func TestPruneNodeModulesAppLayer(t *testing.T) {
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "prune-app",
  "version": "1.0.0",
  "dependencies": { "lodash": "4.17.21" }
}`)
	if code, out := runM(t, projDir, cfgPath, "install"); code != 0 {
		t.Fatalf("install exit=%d out=%s", code, out)
	}
	leftover := filepath.Join(projDir, "node_modules", "leftover-pkg")
	if err := os.MkdirAll(leftover, 0o755); err != nil {
		t.Fatal(err)
	}
	ac, err := app.New(context.Background(), app.Options{CWD: projDir, ConfigPath: cfgPath})
	if err != nil {
		t.Fatal(err)
	}
	res, err := app.PruneNodeModules(context.Background(), ac, app.PruneOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Removed < 1 {
		t.Fatalf("expected removals, got %+v", res)
	}
}
