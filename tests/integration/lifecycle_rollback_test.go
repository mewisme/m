package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/testkit"
)

func setupLifecycleProject(t *testing.T, dep string) (projDir, cfgPath string) {
	t.Helper()
	testkit.CleanEnv(t)
	testkit.EnableLifecycle(t)
	t.Setenv("NO_PROXY", "*")
	t.Setenv("no_proxy", "*")

	reg := testkit.LoadRegistry(t, "lifecycle/registry")
	srv := reg.Start(t)

	projDir = t.TempDir()
	pkgJSON := `{
  "name": "lifecycle-test",
  "version": "1.0.0",
  "dependencies": { "` + dep + `": "1.0.0" }
}`
	if err := os.WriteFile(filepath.Join(projDir, "package.json"), []byte(pkgJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	cfgPath = filepath.Join(projDir, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte(`{"registry":"`+srv.URL+`"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return projDir, cfgPath
}

func TestLifecycleFailingScriptRollsBack(t *testing.T) {
	projDir, cfgPath := setupLifecycleProject(t, "lifecycle-failing")
	marker := filepath.Join(projDir, "node_modules", ".prior-marker")
	if err := os.MkdirAll(filepath.Dir(marker), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(marker, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, out := runM(t, projDir, cfgPath, "trust", "lifecycle-failing"); code != 0 {
		t.Fatalf("trust exit=%d out=%s", code, out)
	}
	code, out := runM(t, projDir, cfgPath, "install")
	if code == 0 {
		t.Fatalf("expected failure, out=%s", out)
	}
	data, err := os.ReadFile(marker)
	if err != nil || string(data) != "keep" {
		t.Fatalf("prior marker missing or changed: %v %q", err, data)
	}
}
