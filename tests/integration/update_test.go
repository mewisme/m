package integration_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/transaction"
)

func TestUpdateTargetedPreservesUnrelated(t *testing.T) {
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "update-test",
  "version": "1.0.0",
  "dependencies": { "pkg-a": "^1.0.0", "lodash": "4.17.21" }
}`)
	if code, out := runM(t, projDir, cfgPath, "install"); code != 0 {
		t.Fatalf("install exit=%d out=%s", code, out)
	}
	lodashBefore, err := os.ReadFile(filepath.Join(projDir, "node_modules", "lodash", "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	code, out := runM(t, projDir, cfgPath, "update", "pkg-a")
	if code != 0 {
		t.Fatalf("update exit=%d out=%s", code, out)
	}
	lodashAfter, err := os.ReadFile(filepath.Join(projDir, "node_modules", "lodash", "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(lodashAfter) != string(lodashBefore) {
		t.Fatal("lodash tree changed after targeted pkg-a update")
	}
}

func TestUpdateDryRunNoMutation(t *testing.T) {
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "update-dry",
  "version": "1.0.0",
  "dependencies": { "pkg-a": "^1.0.0" }
}`)
	if code, _ := runM(t, projDir, cfgPath, "install"); code != 0 {
		t.Fatal("install failed")
	}
	lockBefore, _ := os.ReadFile(filepath.Join(projDir, "m.lock"))
	code, out := runM(t, projDir, cfgPath, "update", "pkg-a", "--dry-run", "--json")
	if code != 0 {
		t.Fatalf("dry-run exit=%d out=%s", code, out)
	}
	if !strings.Contains(out, `"plan"`) {
		t.Fatalf("missing plan in json output: %s", out)
	}
	lockAfter, _ := os.ReadFile(filepath.Join(projDir, "m.lock"))
	if string(lockAfter) != string(lockBefore) {
		t.Fatal("lock changed after dry-run update")
	}
}

func TestUpdateLatestStagesManifestInMemory(t *testing.T) {
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "update-latest",
  "version": "1.0.0",
  "dependencies": { "pkg-a": "1.0.0" }
}`)
	if code, _ := runM(t, projDir, cfgPath, "install"); code != 0 {
		t.Fatal("install failed")
	}
	pkgBefore, _ := os.ReadFile(filepath.Join(projDir, "package.json"))
	code, out := runM(t, projDir, cfgPath, "update", "pkg-a", "--latest", "--dry-run")
	if code != 0 {
		t.Fatalf("dry-run --latest exit=%d out=%s", code, out)
	}
	pkgAfter, err := os.ReadFile(filepath.Join(projDir, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(pkgAfter) != string(pkgBefore) {
		t.Fatal("package.json mutated before commit on --latest --dry-run")
	}
}

func TestUpdateRegistryFailPreservesTree(t *testing.T) {
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "update-reg-fail",
  "version": "1.0.0",
  "dependencies": { "pkg-a": "^1.0.0", "lodash": "4.17.21" }
}`)
	if code, _ := runM(t, projDir, cfgPath, "install"); code != 0 {
		t.Fatal("install failed")
	}
	if err := os.WriteFile(cfgPath, []byte(`{"registry":"http://127.0.0.1:1"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	seed, _ := os.ReadFile(filepath.Join(projDir, "node_modules", "lodash", "package.json"))
	code, _ := runM(t, projDir, cfgPath, "update", "pkg-a")
	if code == 0 {
		t.Fatal("expected registry failure")
	}
	after, err := os.ReadFile(filepath.Join(projDir, "node_modules", "lodash", "package.json"))
	if err != nil || string(after) != string(seed) {
		t.Fatal("tree mutated after registry failure")
	}
}

func TestUpdateCommitFailPreservesLock(t *testing.T) {
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "update-commit-fail",
  "version": "1.0.0",
  "dependencies": { "pkg-a": "^1.0.0", "lodash": "4.17.21" }
}`)
	if code, _ := runM(t, projDir, cfgPath, "install"); code != 0 {
		t.Fatal("install failed")
	}
	lockBefore, _ := os.ReadFile(filepath.Join(projDir, "m.lock"))
	transaction.SetTestHook(func(phase string, opIndex int) error {
		if phase == "publish" && opIndex == 0 {
			return os.ErrPermission
		}
		return nil
	})
	t.Cleanup(func() { transaction.SetTestHook(nil) })
	code, out := runM(t, projDir, cfgPath, "update", "pkg-a")
	if code == 0 {
		t.Fatalf("expected commit failure, out=%s", out)
	}
	lockAfter, _ := os.ReadFile(filepath.Join(projDir, "m.lock"))
	if string(lockAfter) != string(lockBefore) {
		t.Fatal("lock changed after failed update commit")
	}
}

func TestUpdateLinkFailPreservesTree(t *testing.T) {
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "update-link-fail",
  "version": "1.0.0",
  "dependencies": { "pkg-a": "^1.0.0", "lodash": "4.17.21" }
}`)
	if code, _ := runM(t, projDir, cfgPath, "install"); code != 0 {
		t.Fatal("install failed")
	}
	seed, _ := os.ReadFile(filepath.Join(projDir, "node_modules", "lodash", "package.json"))
	transaction.SetTestHook(func(phase string, opIndex int) error {
		if phase == "post_validate" {
			return os.ErrPermission
		}
		return nil
	})
	t.Cleanup(func() { transaction.SetTestHook(nil) })
	code, _ := runM(t, projDir, cfgPath, "update", "pkg-a")
	if code == 0 {
		t.Fatal("expected validate/link failure")
	}
	after, err := os.ReadFile(filepath.Join(projDir, "node_modules", "lodash", "package.json"))
	if err != nil || string(after) != string(seed) {
		t.Fatal("tree mutated after link/validate failure")
	}
}

func TestUpdateFetchFailPreservesTree(t *testing.T) {
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "update-fetch-fail",
  "version": "1.0.0",
  "dependencies": { "pkg-a": "^1.0.0", "lodash": "4.17.21" }
}`)
	if code, _ := runM(t, projDir, cfgPath, "install"); code != 0 {
		t.Fatal("install failed")
	}
	seed, _ := os.ReadFile(filepath.Join(projDir, "node_modules", "lodash", "package.json"))
	if err := os.WriteFile(cfgPath, []byte(`{"registry":"http://127.0.0.1:1"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = os.RemoveAll(filepath.Join(projDir, ".mew", "cache"))
	code, _ := runM(t, projDir, cfgPath, "update", "pkg-a")
	if code == 0 {
		t.Fatal("expected fetch failure")
	}
	after, err := os.ReadFile(filepath.Join(projDir, "node_modules", "lodash", "package.json"))
	if err != nil || string(after) != string(seed) {
		t.Fatal("tree mutated after fetch failure")
	}
}
