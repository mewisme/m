package integration_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/cli"
	"github.com/mewisme/mew/internal/testkit"
)

func TestCiCleanInstall(t *testing.T) {
	projDir := t.TempDir()
	testkit.CopyFixture(t, "projects/ci-clean-install", projDir)
	reg := testkit.LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)
	cfgPath := filepath.Join(projDir, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte(`{"registry":"`+srv.URL+`"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	testkit.CleanEnv(t)
	t.Setenv("NO_PROXY", "*")

	if code, out := runM(t, projDir, cfgPath, "install"); code != 0 {
		t.Fatalf("install exit=%d out=%s", code, out)
	}
	lockBefore, err := os.ReadFile(filepath.Join(projDir, "m.lock"))
	if err != nil {
		t.Fatal(err)
	}
	extraneous := filepath.Join(projDir, "node_modules", "extraneous-pkg")
	if err := os.MkdirAll(extraneous, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(extraneous, "package.json"), []byte(`{"name":"extraneous-pkg","version":"1.0.0"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	code, out := runM(t, projDir, cfgPath, "ci")
	if code != 0 {
		t.Fatalf("ci exit=%d out=%s", code, out)
	}
	if _, err := os.Stat(filepath.Join(projDir, "node_modules", "extraneous-pkg")); !os.IsNotExist(err) {
		t.Fatal("extraneous-pkg should be removed after ci")
	}
	if _, err := os.Stat(filepath.Join(projDir, "node_modules", "pkg-a", "package.json")); err != nil {
		t.Fatal("pkg-a missing after ci:", err)
	}
	lockAfter, err := os.ReadFile(filepath.Join(projDir, "m.lock"))
	if err != nil {
		t.Fatal(err)
	}
	if string(lockAfter) != string(lockBefore) {
		t.Fatal("m.lock changed after ci")
	}
}

func TestCiFrozenDriftFails(t *testing.T) {
	projDir := t.TempDir()
	testkit.CopyFixture(t, "projects/ci-clean-install", projDir)
	reg := testkit.LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)
	cfgPath := filepath.Join(projDir, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte(`{"registry":"`+srv.URL+`"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	testkit.CleanEnv(t)
	t.Setenv("NO_PROXY", "*")

	pkgPath := filepath.Join(projDir, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(string(data), "pkg-a", "pkg-c", 1)
	if err := os.WriteFile(pkgPath, []byte(updated), 0o644); err != nil {
		t.Fatal(err)
	}

	cliRoot := cli.NewMRoot(cli.BuildInfo{Version: "0.0.0-test"})
	errBuf := new(bytes.Buffer)
	cliRoot.SetOut(io.Discard)
	cliRoot.SetErr(errBuf)
	cliRoot.SetArgs([]string{"--cwd", projDir, "--config", cfgPath, "ci"})
	err = cliRoot.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected ci drift failure")
	}
	if apperr.CodeOf(err) != apperr.Lockfile {
		t.Fatalf("code=%s err=%v out=%s", apperr.CodeOf(err), err, errBuf.String())
	}
}

func TestCiRejectsDryRun(t *testing.T) {
	projDir := t.TempDir()
	testkit.CopyFixture(t, "projects/ci-clean-install", projDir)
	reg := testkit.LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)
	cfgPath := filepath.Join(projDir, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte(`{"registry":"`+srv.URL+`"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	testkit.CleanEnv(t)
	t.Setenv("NO_PROXY", "*")

	cliRoot := cli.NewMRoot(cli.BuildInfo{Version: "0.0.0-test"})
	errBuf := new(bytes.Buffer)
	cliRoot.SetOut(io.Discard)
	cliRoot.SetErr(errBuf)
	cliRoot.SetArgs([]string{"--cwd", projDir, "--config", cfgPath, "ci", "--dry-run"})
	err := cliRoot.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected dry-run rejection")
	}
	if apperr.CodeOf(err) != apperr.Usage {
		t.Fatalf("code=%s err=%v", apperr.CodeOf(err), err)
	}
}
