package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/m/internal/testkit"
)

func TestAddScopedPackageUsesProjectRegistryMapping(t *testing.T) {
	testkit.CleanEnv(t)
	t.Setenv("NO_PROXY", "*")
	reg := testkit.LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)
	defer srv.Close()

	projDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projDir, "package.json"), []byte(`{
  "name": "scope-add",
  "version": "1.0.0"
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(projDir, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte(`{
  "registry": "`+srv.URL+`",
  "registries": { "@scope": "`+srv.URL+`" }
}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	code, out := runM(t, projDir, cfgPath, "add", "@scope/pkg")
	if code != 0 {
		t.Fatalf("add @scope/pkg exit=%d out=%s", code, out)
	}
	if _, err := os.Stat(filepath.Join(projDir, "node_modules", "@scope", "pkg", "package.json")); err != nil {
		t.Fatalf("scoped package not linked: %v out=%s", err, out)
	}
}
