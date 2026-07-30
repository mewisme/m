package integration_test

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/cli"
	"github.com/mewisme/mew/internal/registry"
	"github.com/mewisme/mew/internal/testkit"
)

func runMX(t *testing.T, args ...string) (int, string) {
	t.Helper()
	registry.TestProbeReset()
	root := cli.NewMXRoot(cli.BuildInfo{Version: "0.0.0-test"})
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	root.SetOut(outBuf)
	root.SetErr(errBuf)
	root.SetArgs(args)
	code := cli.ExecuteWithArgv(root, context.Background(), args)
	return code, outBuf.String() + errBuf.String()
}

func runMXInProject(t *testing.T, projDir string, args ...string) (int, string) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(projDir); err != nil {
		t.Fatal(err)
	}
	return runMX(t, args...)
}

func TestExecuteMXVersionReserved(t *testing.T) {
	testkit.CleanEnv(t)
	code, _ := runMX(t, "version")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
}

func TestMXParseMissingSelectorUsage(t *testing.T) {
	testkit.CleanEnv(t)
	code, out := runMX(t, "--offline")
	if code != 2 {
		t.Fatalf("exit=%d", code)
	}
	if !strings.Contains(out, "ERR_M_USAGE") {
		t.Fatalf("out=%s", out)
	}
}

func TestMXLocalHitNoRegistryCalls(t *testing.T) {
	skipWithoutNode(t)
	testkit.CleanEnv(t)
	proj := testkit.SetupMXLocalFixture(t, "echo-bin")
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(proj); err != nil {
		t.Fatal(err)
	}
	code, out := runMX(t, "demo-pkg", "--", "hello")
	if code != 0 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	if registry.TestProbeMetadataCalls() != 0 {
		t.Fatalf("metadata calls=%d", registry.TestProbeMetadataCalls())
	}
	if registry.TestProbeArtifactCalls() != 0 {
		t.Fatalf("artifact calls=%d", registry.TestProbeArtifactCalls())
	}
}

func withNonInteractiveStdin(t *testing.T, fn func()) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdin
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = old
		_ = r.Close()
		_ = w.Close()
	})
	_ = w.Close()
	fn()
}

func TestMXNonTTYWithoutYesFails(t *testing.T) {
	testkit.CleanEnv(t)
	proj, _, _ := testkit.SetupMXRegistryProject(t, `{"name":"mx-consent-stub","version":"1.0.0"}`)
	withNonInteractiveStdin(t, func() {
		code, out := runMXInProject(t, proj, "pkg-cli@1.0.0")
		if code != 2 {
			t.Fatalf("exit=%d out=%s", code, out)
		}
		if !strings.Contains(out, "ERR_M_USAGE") {
			t.Fatalf("out=%s", out)
		}
		if registry.TestProbeArtifactCalls() != 0 {
			t.Fatalf("artifact calls=%d", registry.TestProbeArtifactCalls())
		}
	})
}

func TestMXReservedCachePrune(t *testing.T) {
	testkit.CleanEnv(t)
	proj := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(proj); err != nil {
		t.Fatal(err)
	}
	code, out := runMX(t, "cache", "prune", "--dry-run")
	if code != 0 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	if registry.TestProbeMetadataCalls() != 0 || registry.TestProbeArtifactCalls() != 0 {
		t.Fatalf("metadata=%d artifact=%d", registry.TestProbeMetadataCalls(), registry.TestProbeArtifactCalls())
	}
	_ = out
}

func TestMXModeBPackageFlag(t *testing.T) {
	skipWithoutNode(t)
	testkit.CleanEnv(t)
	proj, _, _ := testkit.SetupMXRegistryProject(t, `{"name":"mx-mode-b","version":"1.0.0"}`)
	code, out := runMXInProject(t, proj, "--yes", "-p", "pkg-cli", "cli", "--", "mode-b-ok")
	if code != 0 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	if registry.TestProbeArtifactCalls() == 0 {
		t.Fatalf("expected artifact fetch with --yes")
	}
}
