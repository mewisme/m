package testkit

import (
	"os"
	"path/filepath"
	"testing"
)

// MXFixtureDir returns the mx fixture root.
func MXFixtureDir(name string) string {
	return filepath.Join("fixtures", "mx", name)
}

// SetupMXLocalFixture creates a project where demo-pkg is a declared dependency with one local bin.
func SetupMXLocalFixture(t testing.TB, command string) string {
	t.Helper()
	root := t.TempDir()
	manifest := `{"name":"mx-local-fixture","version":"1.0.0","dependencies":{"demo-pkg":"1.0.0"}}` + "\n"
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	root = setupExecBinAtRoot(t, root, command)
	pkgManifest := `{"name":"demo-pkg","version":"1.0.0","bin":{"` + command + `":"./cli.js"}}` + "\n"
	if err := os.WriteFile(filepath.Join(root, "node_modules", "demo-pkg", "package.json"), []byte(pkgManifest), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// SetupMXRegistryProject creates a temp project with package.json and registry config.
func SetupMXRegistryProject(t testing.TB, packageJSON string) (projDir, cfgPath, srvURL string) {
	t.Helper()
	CleanEnv(t)
	t.Setenv("NO_PROXY", "*")
	t.Setenv("no_proxy", "*")

	reg := LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)

	projDir = t.TempDir()
	if err := os.WriteFile(filepath.Join(projDir, "package.json"), []byte(packageJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	cfgPath = filepath.Join(projDir, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte(`{"registry":"`+srv.URL+`"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return projDir, cfgPath, srv.URL
}
