package integration_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/cli"
	"github.com/mewisme/mew/internal/registry"
	"github.com/mewisme/mew/internal/testkit"
)

func runMExec(t *testing.T, projDir string, args ...string) (int, string) {
	t.Helper()
	registry.TestProbeReset()
	cliRoot := cli.NewMRoot(cli.BuildInfo{Version: "0.0.0-test"})
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	cliRoot.SetOut(outBuf)
	cliRoot.SetErr(errBuf)
	full := append([]string{"--cwd", projDir}, args...)
	cliRoot.SetArgs(full)
	code := cli.ExecuteWithContext(cliRoot, context.Background())
	out := outBuf.String() + errBuf.String()
	return code, out
}

func TestExecDoesNotUseMewRegistryClient(t *testing.T) {
	skipWithoutNode(t)
	testkit.CleanEnv(t)
	proj := testkit.SetupExecFixture(t, "echo-bin")
	code, out := runMExec(t, proj, "exec", "echo-bin", "--", "hello")
	if code != 0 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	if registry.TestProbeCalls() != 0 {
		t.Fatalf("registry calls=%d", registry.TestProbeCalls())
	}
}

func TestDirectBinDispatchDoesNotUseMewRegistryClient(t *testing.T) {
	skipWithoutNode(t)
	testkit.CleanEnv(t)
	testkit.EnableDirectBinDispatch(t)
	proj := testkit.SetupExecFixture(t, "echo-bin")
	code, out := runMExec(t, proj, "echo-bin", "--", "hello")
	if code != 0 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	if registry.TestProbeCalls() != 0 {
		t.Fatalf("registry calls=%d", registry.TestProbeCalls())
	}
}

func TestExecRejectsRecursive(t *testing.T) {
	testkit.CleanEnv(t)
	proj := testkit.SetupExecFixture(t, "echo-bin")
	code, out := runMExec(t, proj, "-r", "exec", "echo-bin")
	if code != 2 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	if !strings.Contains(out, "ERR_M_USAGE") {
		t.Fatalf("out=%s", out)
	}
}

func TestExecFilterZero(t *testing.T) {
	testkit.CleanEnv(t)
	testkit.EnableWorkspaces(t)
	proj := setupDispatchWorkspaceFixture(t, "dag-simple")
	code, out := runMExec(t, proj, "exec", "--filter", "missing", "echo-bin")
	if code != 1 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	if !strings.Contains(out, "ERR_M_NOT_FOUND") {
		t.Fatalf("out=%s", out)
	}
}

func TestExecMissNeverGuessesPackage(t *testing.T) {
	testkit.CleanEnv(t)
	proj := testkit.SetupExecFixture(t, "echo-bin")
	code, out := runMExec(t, proj, "exec", "tsc")
	if code != 1 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	if strings.Contains(out, "m add tsc") {
		t.Fatalf("guessed package from command: %s", out)
	}
	if !strings.Contains(out, "m exec --package") {
		t.Fatalf("out=%s", out)
	}
}

func TestDirectDispatchGateOffHint(t *testing.T) {
	testkit.CleanEnv(t)
	proj := testkit.SetupExecFixture(t, "echo-bin")
	code, out := runMExec(t, proj, "echo-bin")
	if code != 2 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	if !strings.Contains(out, "m exec echo-bin") {
		t.Fatalf("out=%s", out)
	}
}

func TestBinMetaSchemaRoundTripIntegration(t *testing.T) {
	testkit.CleanEnv(t)
	_ = testkit.SetupExecFixture(t, "tool")
}
