package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/cli"
	"github.com/mewisme/mew/internal/registry"
	"github.com/mewisme/mew/internal/testkit"
)

func runMEnvInspect(t *testing.T, projDir string, args ...string) (int, string) {
	t.Helper()
	cliRoot := cli.NewMRoot(cli.BuildInfo{Version: "0.0.0-test"})
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	cliRoot.SetOut(outBuf)
	cliRoot.SetErr(errBuf)
	full := append([]string{"--cwd", projDir}, args...)
	cliRoot.SetArgs(full)
	code := cli.ExecuteWithContext(cliRoot, context.Background())
	return code, outBuf.String() + errBuf.String()
}

func TestUnifiedExecProjectInspect(t *testing.T) {
	testkit.CleanEnv(t)
	proj := testkit.SetupExecFixture(t, "echo-bin")
	code, out := runMEnvInspect(t, proj, "env", "inspect", "project", "echo-bin")
	if code != 0 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &doc); err != nil {
		t.Fatalf("json: %v out=%s", err, out)
	}
	if doc["source"] != "project" {
		t.Fatalf("source=%v", doc["source"])
	}
}

func TestEnvInspectProjectNoNetwork(t *testing.T) {
	testkit.CleanEnv(t)
	registry.TestProbeReset()
	proj := testkit.SetupExecFixture(t, "tool")
	code, out := runMEnvInspect(t, proj, "env", "inspect", "project")
	if code != 0 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	if registry.TestProbeCalls() != 0 {
		t.Fatalf("registry calls=%d", registry.TestProbeCalls())
	}
	_ = out
}

func TestEnvInspectSnapshotMissingNoNetwork(t *testing.T) {
	testkit.CleanEnv(t)
	registry.TestProbeReset()
	proj := testkit.SetupExecFixture(t, "tool")
	code, out := runMEnvInspect(t, proj, "env", "inspect", "snapshot", "missing-id")
	if code == 0 {
		t.Fatalf("expected failure out=%s", out)
	}
	if registry.TestProbeCalls() != 0 {
		t.Fatalf("registry calls=%d", registry.TestProbeCalls())
	}
}

func TestEnvInspectCapsuleMissingNoNetwork(t *testing.T) {
	testkit.CleanEnv(t)
	registry.TestProbeReset()
	proj := testkit.SetupExecFixture(t, "tool")
	code, out := runMEnvInspect(t, proj, "env", "inspect", "capsule", filepath.Join(proj, "missing.mcap"))
	if code == 0 {
		t.Fatalf("expected failure out=%s", out)
	}
	if registry.TestProbeCalls() != 0 {
		t.Fatalf("registry calls=%d", registry.TestProbeCalls())
	}
}

func TestExecSnapshotUsageMissingID(t *testing.T) {
	testkit.CleanEnv(t)
	proj := testkit.SetupExecFixture(t, "tool")
	code, out := runMExec(t, proj, "exec", "--snapshot")
	if code != 2 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	if !strings.Contains(out, "ERR_M_USAGE") {
		t.Fatalf("out=%s", out)
	}
}

func TestExecCapsuleUsageMissingPath(t *testing.T) {
	testkit.CleanEnv(t)
	proj := testkit.SetupExecFixture(t, "tool")
	code, out := runMExec(t, proj, "exec", "--capsule")
	if code != 2 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	if !strings.Contains(out, "ERR_M_USAGE") {
		t.Fatalf("out=%s", out)
	}
}

func TestExecSnapshotChildArgPassthrough(t *testing.T) {
	testkit.CleanEnv(t)
	proj := testkit.SetupExecFixture(t, "tool")
	code, out := runMExec(t, proj, "exec", "--snapshot", "missing", "tool", "--snapshot", "child")
	if code == 0 {
		t.Fatalf("expected failure for missing snapshot, out=%s", out)
	}
	if strings.Contains(out, "child") && strings.Contains(out, "ERR_M_USAGE") {
		t.Fatalf("child flag leaked into parser: %s", out)
	}
}

func TestUnifiedExecFixtureLayout(t *testing.T) {
	root := testkit.FixtureDir(t, "unified-exec/project")
	if _, err := os.Stat(filepath.Join(root, "package.json")); err != nil {
		t.Fatalf("fixture missing: %v", err)
	}
}
