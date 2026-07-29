package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mewisme/mew/internal/app"
)

func TestPlanMatchesInstallDryRunJSON(t *testing.T) {
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "plan-match",
  "version": "1.0.0",
  "dependencies": { "pkg-a": "^1.0.0" }
}`)
	planCode, planOut := runM(t, projDir, cfgPath, "plan", "--json")
	if planCode != 0 {
		t.Fatalf("plan exit=%d out=%s", planCode, planOut)
	}
	installCode, installOut := runM(t, projDir, cfgPath, "install", "--dry-run", "--json")
	if installCode != 0 {
		t.Fatalf("install dry-run exit=%d out=%s", installCode, installOut)
	}
	var planRes, installRes app.InstallResult
	if err := json.Unmarshal([]byte(planOut), &planRes); err != nil {
		t.Fatalf("plan json: %v\n%s", err, planOut)
	}
	if err := json.Unmarshal([]byte(installOut), &installRes); err != nil {
		t.Fatalf("install json: %v\n%s", err, installOut)
	}
	if planRes.Added != installRes.Added || planRes.Removed != installRes.Removed ||
		planRes.Changed != installRes.Changed || planRes.Packages != installRes.Packages {
		t.Fatalf("count mismatch plan=%+v install=%+v", planRes, installRes)
	}
	if planRes.Plan == nil || installRes.Plan == nil {
		t.Fatal("missing plan in json output")
	}
	planJSON, err := json.Marshal(planRes.Plan)
	if err != nil {
		t.Fatal(err)
	}
	installJSON, err := json.Marshal(installRes.Plan)
	if err != nil {
		t.Fatal(err)
	}
	if string(planJSON) != string(installJSON) {
		t.Fatalf("plan mismatch\nplan=%s\ninstall=%s", planJSON, installJSON)
	}
}

func TestPlanNoDiskMutation(t *testing.T) {
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "plan-nomut",
  "version": "1.0.0",
  "dependencies": { "pkg-a": "^1.0.0", "lodash": "4.17.21" }
}`)
	pkgPath := filepath.Join(projDir, "package.json")
	pkgInfo, err := os.Stat(pkgPath)
	if err != nil {
		t.Fatal(err)
	}
	code, out := runM(t, projDir, cfgPath, "plan", "--json")
	if code != 0 {
		t.Fatalf("plan exit=%d out=%s", code, out)
	}
	if !strings.Contains(out, `"plan"`) {
		t.Fatalf("missing plan in output: %s", out)
	}
	after, err := os.Stat(pkgPath)
	if err != nil {
		t.Fatal(err)
	}
	if !pkgInfo.ModTime().Equal(after.ModTime()) {
		t.Fatal("package.json mtime changed after plan")
	}
	if _, err := os.Stat(filepath.Join(projDir, "m.lock")); err == nil {
		lockBefore, _ := os.Stat(filepath.Join(projDir, "m.lock"))
		time.Sleep(10 * time.Millisecond)
		lockAfter, _ := os.Stat(filepath.Join(projDir, "m.lock"))
		if lockBefore != nil && lockAfter != nil && !lockBefore.ModTime().Equal(lockAfter.ModTime()) {
			t.Fatal("m.lock mtime changed after plan")
		}
	}
	if _, err := os.Stat(filepath.Join(projDir, "node_modules")); !os.IsNotExist(err) {
		t.Fatal("node_modules should not exist after plan")
	}
}

func TestPlanOutputWritesFile(t *testing.T) {
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "plan-output",
  "version": "1.0.0",
  "dependencies": { "pkg-a": "^1.0.0" }
}`)
	outPath := filepath.Join(projDir, "plan-out.json")
	code, cliOut := runM(t, projDir, cfgPath, "plan", "--output", outPath)
	if code != 0 {
		t.Fatalf("plan exit=%d out=%s", code, cliOut)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"schemaVersion"`) {
		t.Fatalf("output missing plan schema: %s", data)
	}
}

func TestPlanInstallDeltaGolden(t *testing.T) {
	goldenDir := filepath.Join("..", "..", "testdata", "plan", "install-delta-golden")
	manifest, err := os.ReadFile(filepath.Join(goldenDir, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	projDir, cfgPath, _ := setupRegistryProject(t, string(manifest))
	code, out := runM(t, projDir, cfgPath, "plan", "--json")
	if code != 0 {
		t.Fatalf("plan exit=%d out=%s", code, out)
	}
	var res app.InstallResult
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatal(err)
	}
	if res.Added == 0 || res.Packages == 0 {
		t.Fatalf("expected non-empty install plan: %+v", res)
	}
	if res.Plan == nil {
		t.Fatal("missing plan")
	}
	for _, op := range res.Plan.Operations {
		switch op.Op {
		case "fetch", "link", "unlink", "script":
		default:
			t.Fatalf("unexpected op %q", op.Op)
		}
	}
}

func TestPlanUpdatePreview(t *testing.T) {
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "plan-update",
  "version": "1.0.0",
  "dependencies": { "pkg-a": "^1.0.0" }
}`)
	if code, out := runM(t, projDir, cfgPath, "install"); code != 0 {
		t.Fatalf("install exit=%d out=%s", code, out)
	}
	lockBefore, _ := os.ReadFile(filepath.Join(projDir, "m.lock"))
	code, out := runM(t, projDir, cfgPath, "plan", "update", "pkg-a", "--json")
	if code != 0 {
		t.Fatalf("plan update exit=%d out=%s", code, out)
	}
	if !strings.Contains(out, `"plan"`) {
		t.Fatalf("missing plan: %s", out)
	}
	lockAfter, _ := os.ReadFile(filepath.Join(projDir, "m.lock"))
	if string(lockAfter) != string(lockBefore) {
		t.Fatal("lock changed after plan update")
	}
}
