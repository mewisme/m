package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/testkit"
)

func testPnpmProjectWithLock(t *testing.T, lockYAML string) (*Context, string) {
	t.Helper()
	testkit.CleanEnv(t)
	home := t.TempDir()
	proj := filepath.Join(home, "proj")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	pkg := `{"name":"migrate-loss","version":"1.0.0","private":true,"packageManager":"pnpm@9.0.0"}`
	if err := os.WriteFile(filepath.Join(proj, "package.json"), []byte(pkg), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(proj, "pnpm-lock.yaml"), []byte(lockYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	loadOpts := config.LoadOptions{CWD: proj, ProjectRoot: proj}
	eff, err := config.Load(context.Background(), loadOpts)
	if err != nil {
		t.Fatal(err)
	}
	return &Context{CWD: proj, Config: eff, ConfigLoadSpec: config.LoadSpecFromOptions(loadOpts)}, proj
}

func TestMigratePnpmDryRunReportsEnginesLoss(t *testing.T) {
	ac, _ := testPnpmProjectWithLock(t, testPnpmLockYAML)
	result, err := MigrateLock(context.Background(), ac, MigrateLockOptions{From: "pnpm", DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range result.LossReport.Items {
		if strings.Contains(item.Field, ".engines") {
			found = true
			if item.Semantic {
				t.Fatalf("engines should be preserved in extension, not semantic: %+v", item)
			}
			if item.Category != "extension" {
				t.Fatalf("category=%q want extension", item.Category)
			}
			break
		}
	}
	if !found {
		t.Fatalf("loss=%v", result.LossReport.Items)
	}
}

func TestMigratePnpmPreservesEnginesInExtension(t *testing.T) {
	ac, proj := testPnpmProjectWithLock(t, testPnpmLockYAML)
	result, err := MigrateLock(context.Background(), ac, MigrateLockOptions{From: "pnpm", DryRun: false, PnpmMajor: 9})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(proj, "pnpm-lock.yaml")); !os.IsNotExist(err) {
		t.Fatal("source pnpm-lock.yaml must be removed after migrate")
	}
	data, err := os.ReadFile(filepath.Join(proj, "m.lock"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), mewMigratePnpmExt) {
		t.Fatalf("m.lock missing %s extension; path=%s", mewMigratePnpmExt, result.Path)
	}
	if !strings.Contains(string(data), `"engines"`) {
		t.Fatal("m.lock extension must retain package engines")
	}
}
