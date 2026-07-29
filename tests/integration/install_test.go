package integration_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/cli"
	"github.com/mewisme/mew/internal/testkit"
)

func setupRegistryProject(t *testing.T, packageJSON string) (projDir, cfgPath, srvURL string) {
	t.Helper()
	testkit.CleanEnv(t)
	t.Setenv("NO_PROXY", "*")
	t.Setenv("no_proxy", "*")

	reg := testkit.LoadRegistry(t, "registry/v1")
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

func runM(t *testing.T, projDir, cfgPath string, args ...string) (int, string) {
	t.Helper()
	cliRoot := cli.NewMRoot(cli.BuildInfo{Version: "0.0.0-test"})
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	cliRoot.SetOut(outBuf)
	cliRoot.SetErr(errBuf)
	full := append([]string{"--cwd", projDir, "--config", cfgPath}, args...)
	cliRoot.SetArgs(full)
	code := cli.ExecuteWithContext(cliRoot, context.Background())
	out := outBuf.String()
	errOut := errBuf.String()
	if code != 0 {
		trimmed := strings.TrimSpace(out)
		if trimmed != "" && strings.HasPrefix(trimmed, "{") {
			return code, out
		}
		if out != "" && errOut != "" {
			return code, out + errOut
		}
		if errOut != "" {
			return code, errOut
		}
	}
	if out != "" {
		return code, out
	}
	return code, errOut
}

func TestInstallGreenfieldBasicCJS(t *testing.T) {
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "basic-cjs",
  "version": "1.0.0",
  "dependencies": { "lodash": "4.17.21" }
}`)
	code, out := runM(t, projDir, cfgPath, "install")
	if code != 0 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	lodashNM := filepath.Join(projDir, "node_modules", "lodash", "package.json")
	if _, err := os.Stat(lodashNM); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(projDir, "m.lock")); err != nil {
		t.Fatal("m.lock missing")
	}
	if node, err := exec.LookPath("node"); err == nil {
		cmd := exec.Command(node, "-e", "require('lodash')")
		cmd.Dir = projDir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("node require: %v\n%s", err, out)
		}
	}
}

func TestInstallFromLockHints(t *testing.T) {
	projDir := t.TempDir()
	testkit.CopyFixture(t, "projects/mlock-greenfield", projDir)
	reg := testkit.LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)
	cfgPath := filepath.Join(projDir, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte(`{"registry":"`+srv.URL+`"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	testkit.CleanEnv(t)
	t.Setenv("NO_PROXY", "*")

	code, out := runM(t, projDir, cfgPath, "install")
	if code != 0 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	for _, pkg := range []string{"pkg-a", "pkg-b"} {
		if _, err := os.Stat(filepath.Join(projDir, "node_modules", pkg, "package.json")); err != nil {
			t.Fatalf("%s: %v", pkg, err)
		}
	}
}

func TestInstallFrozenDriftFails(t *testing.T) {
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "frozen-test",
  "version": "1.0.0",
  "dependencies": { "pkg-a": "^1.0.0" }
}`)
	code, _ := runM(t, projDir, cfgPath, "install")
	if code != 0 {
		t.Fatalf("initial install exit=%d", code)
	}
	pkgPath := filepath.Join(projDir, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(string(data), "pkg-a", "pkg-c", 1)
	if err := os.WriteFile(pkgPath, []byte(updated), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out := runM(t, projDir, cfgPath, "install", "--frozen-lockfile")
	if code == 0 {
		t.Fatalf("expected frozen failure, out=%s", out)
	}
}

func TestInstallDryRunNoMutation(t *testing.T) {
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "dry-run",
  "version": "1.0.0",
  "dependencies": { "lodash": "4.17.21" }
}`)
	code, out := runM(t, projDir, cfgPath, "install", "--dry-run")
	if code != 0 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	if !strings.Contains(out, "dry-run:") {
		t.Fatalf("missing dry-run prefix: %s", out)
	}
	if _, err := os.Stat(filepath.Join(projDir, "node_modules")); !os.IsNotExist(err) {
		t.Fatal("node_modules should not exist after dry-run")
	}
}

func TestAddUpdatesLockAndTree(t *testing.T) {
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "add-test",
  "version": "1.0.0",
  "dependencies": { "pkg-a": "^1.0.0" }
}`)
	if code, out := runM(t, projDir, cfgPath, "install"); code != 0 {
		t.Fatalf("install exit=%d out=%s", code, out)
	}
	code, out := runM(t, projDir, cfgPath, "add", "pkg-c")
	if code != 0 {
		t.Fatalf("add exit=%d out=%s", code, out)
	}
	if _, err := os.Stat(filepath.Join(projDir, "node_modules", "pkg-c", "package.json")); err != nil {
		t.Fatal(err)
	}
}

func TestRemovePrunesTree(t *testing.T) {
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "remove-test",
  "version": "1.0.0",
  "dependencies": { "lodash": "4.17.21", "pkg-c": "^1.0.0" }
}`)
	if code, out := runM(t, projDir, cfgPath, "install"); code != 0 {
		t.Fatalf("install exit=%d out=%s", code, out)
	}
	code, out := runM(t, projDir, cfgPath, "remove", "pkg-c")
	if code != 0 {
		t.Fatalf("remove exit=%d out=%s", code, out)
	}
	if _, err := os.Stat(filepath.Join(projDir, "node_modules", "pkg-c")); !os.IsNotExist(err) {
		t.Fatal("pkg-c should be removed from node_modules")
	}
	if _, err := os.Stat(filepath.Join(projDir, "node_modules", "lodash", "package.json")); err != nil {
		t.Fatal("lodash should remain")
	}
}

func TestScopedDepsInstall(t *testing.T) {
	projDir := t.TempDir()
	testkit.CopyFixture(t, "projects/scoped-deps", projDir)
	reg := testkit.LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)
	cfgPath := filepath.Join(projDir, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte(`{"registry":"`+srv.URL+`"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	testkit.CleanEnv(t)
	t.Setenv("NO_PROXY", "*")

	code, out := runM(t, projDir, cfgPath, "install")
	if code != 0 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	scopePath := filepath.Join(projDir, "node_modules", "@scope", "pkg", "package.json")
	if _, err := os.Stat(scopePath); err != nil {
		t.Fatal(err)
	}
}

func TestBinShimsCreated(t *testing.T) {
	projDir := t.TempDir()
	testkit.CopyFixture(t, "projects/bin-shims", projDir)
	reg := testkit.LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)
	cfgPath := filepath.Join(projDir, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte(`{"registry":"`+srv.URL+`"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	testkit.CleanEnv(t)
	t.Setenv("NO_PROXY", "*")

	code, out := runM(t, projDir, cfgPath, "install")
	if code != 0 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	binDir := filepath.Join(projDir, "node_modules", ".bin")
	if _, err := os.Stat(filepath.Join(binDir, "cli")); err == nil {
		return
	}
	if _, err := os.Stat(filepath.Join(binDir, "cli.cmd")); err == nil {
		return
	}
	t.Fatalf("no cli shim in %s", binDir)
}

func TestInstallFailurePreservesOldTree(t *testing.T) {
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "fail-test",
  "version": "1.0.0",
  "dependencies": { "lodash": "4.17.21" }
}`)
	if code, _ := runM(t, projDir, cfgPath, "install"); code != 0 {
		t.Fatal("initial install failed")
	}
	seed := filepath.Join(projDir, "node_modules", "lodash", "package.json")
	before, err := os.ReadFile(seed)
	if err != nil {
		t.Fatal(err)
	}
	// Corrupt lock integrity by editing m.lock packages after resolve would be complex;
	// instead break package.json to point at nonexistent version for add attempt.
	code, _ := runM(t, projDir, cfgPath, "add", "nonexistent-pkg-xyz@1.0.0")
	if code == 0 {
		t.Fatal("expected add failure")
	}
	after, err := os.ReadFile(seed)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("existing node_modules mutated after failed add")
	}
}
