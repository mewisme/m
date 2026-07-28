package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/testkit"
)

func testPnpmProjectWithPM(t *testing.T, pm string) (*Context, string) {
	t.Helper()
	testkit.CleanEnv(t)
	t.Setenv("NO_PROXY", "*")
	reg := testkit.LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)
	t.Cleanup(srv.Close)

	home := t.TempDir()
	proj := filepath.Join(home, "proj")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	pkg := `{
  "name": "pnpm-hints-test",
  "version": "1.0.0",
  "private": true,
  "packageManager": "` + pm + `",
  "dependencies": { "lodash": "4.17.21" }
}`
	if err := os.WriteFile(filepath.Join(proj, "package.json"), []byte(pkg), 0o644); err != nil {
		t.Fatal(err)
	}
	fixture := filepath.Join(testkit.ModuleRoot(t), "fixtures", "locks", "generated", "pnpm-9", "basic", "pnpm-lock.yaml")
	data, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(proj, "pnpm-lock.yaml"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	loadOpts := config.LoadOptions{
		CWD:         proj,
		ProjectRoot: proj,
		CLI:         map[string]any{"registry": srv.URL},
	}
	eff, err := config.Load(context.Background(), loadOpts)
	if err != nil {
		t.Fatal(err)
	}
	return &Context{CWD: proj, Config: eff, ConfigLoadSpec: config.LoadSpecFromOptions(loadOpts)}, proj
}

func TestValidateIncumbentLockUsesPackageManagerHint(t *testing.T) {
	ac, _ := testPnpmProjectWithPM(t, "pnpm@10.4.0")
	result, err := ValidateIncumbentLock(context.Background(), ac, ValidateLockOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Detection.ProducerMajor != 10 {
		t.Fatalf("det=%+v", result.Detection)
	}
}

func TestValidateIncumbentLockRejectsUnsupportedPMField(t *testing.T) {
	ac, _ := testPnpmProjectWithPM(t, "pnpm@8.0.0")
	_, err := ValidateIncumbentLock(context.Background(), ac, ValidateLockOptions{})
	if err == nil {
		t.Fatal("expected unsupported pnpm@8 error")
	}
	if apperr.CodeOf(err) != apperr.Usage {
		t.Fatalf("code=%s err=%v", apperr.CodeOf(err), err)
	}
}

func TestValidateIncumbentLockFlagConflict(t *testing.T) {
	ac, _ := testPnpmProjectWithPM(t, "pnpm@9.0.0")
	_, err := ValidateIncumbentLock(context.Background(), ac, ValidateLockOptions{PnpmMajor: 10})
	if err == nil {
		t.Fatal("expected conflict")
	}
}

func TestMigrateLockDetectsWithHints(t *testing.T) {
	ac, _ := testPnpmProjectWithPM(t, "pnpm@9.0.0")
	result, err := MigrateLock(context.Background(), ac, MigrateLockOptions{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.Detection.ProducerMajor != 9 {
		t.Fatalf("det=%+v", result.Detection)
	}
}
