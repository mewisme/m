package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
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
		if strings.Contains(item.Field, ".engines") && item.Semantic {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("loss=%v", result.LossReport.Items)
	}
}

func TestMigratePnpmFailsOnEnginesLoss(t *testing.T) {
	ac, proj := testPnpmProjectWithLock(t, testPnpmLockYAML)
	before, err := os.ReadFile(filepath.Join(proj, "pnpm-lock.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = MigrateLock(context.Background(), ac, MigrateLockOptions{From: "pnpm", DryRun: false})
	if err == nil {
		t.Fatal("expected lossy migration failure")
	}
	if apperr.CodeOf(err) != apperr.LockUnrepresentable {
		t.Fatalf("code=%s err=%v", apperr.CodeOf(err), err)
	}
	after, err := os.ReadFile(filepath.Join(proj, "pnpm-lock.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("incumbent lock must be unchanged on migration failure")
	}
	if _, err := os.Stat(filepath.Join(proj, "m.lock")); err == nil {
		t.Fatal("m.lock must not be created on migration failure")
	}
}
