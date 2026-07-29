package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/diagnostics"
	"github.com/mewisme/mew/internal/testkit"
)

const soakIterations = 5

func TestSoakRepresentativeProjects(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping soak test in short mode")
	}

	moduleRoot := testkit.ModuleRoot(t)

	t.Run("greenfield-mlock", func(t *testing.T) {
		soakBenchFixture(t, moduleRoot, "fixtures/soak/representative-projects/greenfield-mlock")
	})
	t.Run("transitive-medium", func(t *testing.T) {
		soakBenchFixture(t, moduleRoot, "fixtures/soak/representative-projects/transitive-medium")
	})
	t.Run("workspace-two-pkg", func(t *testing.T) {
		soakWorkspaceFixture(t)
	})
}

func soakBenchFixture(t *testing.T, moduleRoot, fixture string) {
	t.Helper()
	testkit.CleanEnv(t)

	ctx := context.Background()
	ac, err := app.New(ctx, app.Options{
		CWD:      moduleRoot,
		Reporter: diagnostics.NewReporter(diagnostics.Options{Format: "silent"}),
		Version:  "test",
	})
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < soakIterations; i++ {
		mode := app.BenchWarm
		if i == 0 {
			mode = app.BenchCold
		}
		if _, err := app.BenchInstall(ctx, ac, app.BenchInstallOptions{
			Fixture: fixture,
			Mode:    mode,
		}); err != nil {
			t.Fatalf("iteration %d: %v", i+1, err)
		}
	}
}

func soakWorkspaceFixture(t *testing.T) {
	t.Helper()
	testkit.CleanEnv(t)
	testkit.EnableWorkspaces(t)
	t.Setenv("NO_PROXY", "*")
	t.Setenv("no_proxy", "*")

	reg := testkit.LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)

	projDir := t.TempDir()
	testkit.CopyFixture(t, "soak/representative-projects/workspace-two-pkg", projDir)
	cfgPath := filepath.Join(projDir, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte(`{"registry":"`+srv.URL+`"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	code, out := runM(t, projDir, cfgPath, "install")
	if code != 0 {
		t.Fatalf("initial install exit=%d out=%s", code, out)
	}

	for i := 1; i < soakIterations; i++ {
		code, out := runM(t, projDir, cfgPath, "--filter", "app", "install")
		if code != 0 {
			t.Fatalf("iteration %d filter install exit=%d out=%s", i+1, code, out)
		}
	}
}
