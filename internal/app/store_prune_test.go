package app_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/m/internal/app"
	"github.com/mewisme/m/internal/config"
	"github.com/mewisme/m/internal/testkit"
)

func TestDefaultStoreScanRootsFromSnapshot(t *testing.T) {
	testkit.CleanEnv(t)
	t.Setenv("MEW_HOME", filepath.Join(t.TempDir(), "ambient"))
	proj := filepath.Join(t.TempDir(), "proj")
	homeA := filepath.Join(t.TempDir(), "home-a")
	snap := config.NewEnvSnapshot([]string{"MEW_HOME=" + homeA}, "linux")
	roots := app.DefaultStoreScanRoots(snap, proj)
	if len(roots) != 2 {
		t.Fatalf("roots=%v want 2", roots)
	}
	projAbs, _ := filepath.Abs(proj)
	homeAbs, _ := filepath.Abs(homeA)
	if roots[0] != projAbs {
		t.Fatalf("roots[0]=%q want %q", roots[0], projAbs)
	}
	if roots[1] != homeAbs {
		t.Fatalf("roots[1]=%q want %q", roots[1], homeAbs)
	}
}

func TestDefaultStoreScanRootsDedupesProjectAndHome(t *testing.T) {
	home := t.TempDir()
	snap := config.NewEnvSnapshot([]string{"MEW_HOME=" + home}, "linux")
	roots := app.DefaultStoreScanRoots(snap, home)
	if len(roots) != 1 {
		t.Fatalf("roots=%v want deduped single entry", roots)
	}
}

func TestDefaultStoreScanRootsSkipsEmpty(t *testing.T) {
	snap := config.NewEnvSnapshot([]string{}, "linux")
	if roots := app.DefaultStoreScanRoots(snap, ""); len(roots) != 0 {
		t.Fatalf("roots=%v want empty", roots)
	}
}

func TestPruneStoreDefaultRootsUseSnapshotNotAmbient(t *testing.T) {
	testkit.CleanEnv(t)
	homeA := t.TempDir()
	homeB := t.TempDir()
	t.Setenv("MEW_HOME", homeB)

	storeDir := filepath.Join(homeA, "store")
	projA := filepath.Join(homeA, "proj")
	if err := os.MkdirAll(filepath.Join(projA, ".mew"), 0o755); err != nil {
		t.Fatal(err)
	}
	integrity := "sha512-abc"
	manifest := `{"schemaVersion":1,"packages":["` + integrity + `"]}`
	if err := os.WriteFile(filepath.Join(projA, ".mew", "store-manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	// homeB has no manifest referencing the package.
	if err := os.MkdirAll(filepath.Join(homeB, ".mew"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(homeB, ".mew", "store-manifest.json"), []byte(`{"schemaVersion":1,"packages":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	pkgPath := filepath.Join(storeDir, "packages", "sha512", "abc")
	if err := os.MkdirAll(pkgPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgPath, "package.json"), []byte(`{"name":"orphan"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	eff, err := config.Load(context.Background(), config.LoadOptions{
		CWD:         projA,
		ProjectRoot: projA,
		Env:         []string{"MEW_HOME=" + homeA, "MEW_STORE_DIR=" + storeDir, "MEW_EXPERIMENTAL_GLOBAL_STORE=1"},
		EnvSnapshot: config.NewEnvSnapshot([]string{"MEW_HOME=" + homeA, "MEW_STORE_DIR=" + storeDir, "MEW_EXPERIMENTAL_GLOBAL_STORE=1"}, "linux"),
	})
	if err != nil {
		t.Fatal(err)
	}
	ac := &app.Context{Config: eff}

	dry, err := app.PruneStore(context.Background(), ac, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if dry.Removed != 0 {
		t.Fatalf("dry-run removed=%d want 0 (referenced from snapshot home A)", dry.Removed)
	}
	real, err := app.PruneStore(context.Background(), ac, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if real.Removed != 0 {
		t.Fatalf("prune removed=%d want 0", real.Removed)
	}
	if _, err := os.Stat(pkgPath); err != nil {
		t.Fatalf("referenced package removed: %v", err)
	}
}
