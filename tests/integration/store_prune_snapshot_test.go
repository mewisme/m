package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/testkit"
)

func TestStorePruneSnapshotHomeNotAmbient(t *testing.T) {
	enableGlobalStore(t)
	testkit.CleanEnv(t)
	t.Setenv("NO_PROXY", "*")

	homeA := t.TempDir()
	homeB := t.TempDir()
	t.Setenv("MEW_HOME", homeB)

	reg := testkit.LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)

	projA := filepath.Join(homeA, "proj")
	if err := os.MkdirAll(projA, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projA, "package.json"), []byte(`{
  "name": "prune-snap",
  "version": "1.0.0",
  "dependencies": { "lodash": "4.17.21" }
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(projA, "m.jsonc")
	storeDir := filepath.Join(homeA, "store")
	cfg := `{"registry":"` + srv.URL + `","store.dir":"` + strings.ReplaceAll(storeDir, `\`, `\\`) + `","link.use_global_store":true}`
	if err := os.WriteFile(cfgPath, []byte(cfg+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	env := []string{
		"MEW_HOME=" + homeA,
		"MEW_STORE_DIR=" + storeDir,
		"MEW_EXPERIMENTAL_GLOBAL_STORE=1",
		"NO_PROXY=*",
	}
	ac, err := app.New(context.Background(), app.Options{CWD: projA, ConfigPath: cfgPath, Env: env})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.Install(context.Background(), ac, app.InstallOptions{}); err != nil {
		t.Fatal(err)
	}

	// Ambient home B has an empty manifest — must not be authoritative for prune scan.
	if err := os.MkdirAll(filepath.Join(homeB, ".mew"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(homeB, ".mew", "store-manifest.json"), []byte(`{"schemaVersion":1,"packages":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	dry, err := app.PruneStore(context.Background(), ac, true, app.DefaultStoreScanRoots(ac.Config.Env, projA))
	if err != nil {
		t.Fatal(err)
	}
	if dry.Removed != 0 {
		t.Fatalf("dry-run removed=%d want 0: %v", dry.Removed, dry.Paths)
	}
	real, err := app.PruneStore(context.Background(), ac, false, app.DefaultStoreScanRoots(ac.Config.Env, projA))
	if err != nil {
		t.Fatal(err)
	}
	if real.Removed != 0 {
		t.Fatalf("prune removed=%d want 0", real.Removed)
	}
	entries, err := os.ReadDir(filepath.Join(storeDir, "packages", "sha256"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("referenced lodash package was pruned")
	}
}
