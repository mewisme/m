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
	"github.com/mewisme/mew/internal/testkit"
)

func setupDispatchFixture(t *testing.T, rel string) string {
	t.Helper()
	testkit.CleanEnv(t)
	testkit.EnableDirectScripts(t)
	projDir := t.TempDir()
	testkit.CopyFixture(t, "dispatch/"+rel, projDir)
	return projDir
}

func setupDispatchWorkspaceFixture(t *testing.T, rel string) string {
	t.Helper()
	testkit.CleanEnv(t)
	testkit.EnableDirectScripts(t)
	testkit.EnableWorkspaces(t)
	projDir := t.TempDir()
	testkit.CopyFixture(t, "workspace-runner/"+rel, projDir)
	return projDir
}

func runMProjectArgv(t *testing.T, projDir string, args ...string) (int, string) {
	t.Helper()
	cliRoot := cli.NewMRoot(cli.BuildInfo{Version: "0.0.0-test"})
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	cliRoot.SetOut(outBuf)
	cliRoot.SetErr(errBuf)
	full := append([]string{"--cwd", projDir}, args...)
	cliRoot.SetArgs(full)
	code := cli.ExecuteWithContext(cliRoot, context.Background())
	out := outBuf.String()
	errOut := errBuf.String()
	if code != 0 {
		if trimmed := strings.TrimSpace(out); trimmed != "" && strings.HasPrefix(trimmed, "{") {
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

func TestDispatchDirectBuildArgv(t *testing.T) {
	skipWithoutNode(t)
	projDir := setupDispatchFixture(t, "arg-ambiguity")

	cases := []struct {
		name string
		args []string
		want string
	}{
		{"mode-flag", []string{"build", "--mode", "x"}, `["--mode","x"]`},
		{"separator", []string{"build", "--", "--mode", "x"}, `["--mode","x"]`},
		{"reporter-before", []string{"--reporter", "silent", "build", "--mode", "x"}, `["--mode","x"]`},
		{"reporter-forwarded", []string{"build", "--reporter", "ndjson"}, `["--reporter","ndjson"]`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			testkit.CleanEnv(t)
			testkit.EnableDirectScripts(t)
			argvOut := filepath.Join(projDir, "argv.out")
			_ = os.Remove(argvOut)
			code, out := runMProjectArgv(t, projDir, tc.args...)
			if code != 0 {
				t.Fatalf("exit=%d out=%s", code, out)
			}
			raw, err := os.ReadFile(argvOut)
			if err != nil {
				t.Fatalf("read argv.out: %v out=%s", err, out)
			}
			got := strings.TrimSpace(string(raw))
			if got != tc.want {
				t.Fatalf("argv=%q want %q", got, tc.want)
			}
		})
	}
}

func TestDispatchWorkspaceFlagBeforeSelector(t *testing.T) {
	skipWithoutNode(t)
	projDir := setupDispatchWorkspaceFixture(t, "dag-simple")
	code, out := runMProjectArgv(t, projDir, "-r", "--workspace-concurrency", "2", "build")
	if code != 0 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	_ = out
}

func TestDispatchWorkspaceFlagForwardedToScript(t *testing.T) {
	skipWithoutNode(t)
	projDir := setupDispatchFixture(t, "arg-ambiguity")
	testkit.EnableWorkspaces(t)
	code, out := runMProjectArgv(t, projDir, "-r", "build", "--workspace-concurrency", "2")
	if code == 0 {
		raw, err := os.ReadFile(filepath.Join(projDir, "argv.out"))
		if err != nil {
			t.Fatal(err)
		}
		got := strings.TrimSpace(string(raw))
		if got != `["--workspace-concurrency","2"]` {
			t.Fatalf("argv=%q", got)
		}
		return
	}
	if !strings.Contains(out, "workspaces not enabled") && !strings.Contains(out, "ERR_M") {
		t.Fatalf("exit=%d out=%s", code, out)
	}
}

func TestDispatchBuiltinBeatsScript(t *testing.T) {
	skipWithoutNode(t)
	projDir := setupDispatchFixture(t, "collision-matrix")
	scriptOut := filepath.Join(projDir, "script.out")
	_ = os.Remove(scriptOut)

	code, _ := runMProjectArgv(t, projDir, "add")
	if data, err := os.ReadFile(scriptOut); err == nil && strings.Contains(string(data), "add-script") {
		t.Fatalf("direct m add ran script (exit=%d), want built-in", code)
	}

	_ = os.Remove(scriptOut)
	code, _ = runMProjectArgv(t, projDir, "run", "add")
	if code != 0 {
		t.Fatalf("exit=%d want m run add", code)
	}
	data, err := os.ReadFile(scriptOut)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "add-script") {
		t.Fatalf("out=%s", data)
	}
}

func TestDispatchGateOffRejectsDirect(t *testing.T) {
	skipWithoutNode(t)
	testkit.CleanEnv(t)
	projDir := t.TempDir()
	testkit.CopyFixture(t, "dispatch/collision-matrix", projDir)
	code, out := runMProject(t, projDir, "dev")
	if code != 2 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	if !strings.Contains(out, "Direct script shortcuts are disabled") {
		t.Fatalf("out=%s", out)
	}
}

func TestDispatchUnknownOutsideProject(t *testing.T) {
	testkit.CleanEnv(t)
	testkit.EnableDirectScripts(t)
	projDir := t.TempDir()
	code, out := runMProject(t, projDir, "not-a-command")
	if code != 2 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	if !strings.Contains(out, "ERR_M_USAGE") {
		t.Fatalf("out=%s", out)
	}
}

func TestDispatchExplicitMissingProjectCWD(t *testing.T) {
	testkit.CleanEnv(t)
	testkit.EnableDirectScripts(t)
	projDir := t.TempDir()
	code, out := runMProject(t, projDir, "dev")
	if code != 1 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	if !strings.Contains(out, "ERR_M_NOT_FOUND") {
		t.Fatalf("out=%s", out)
	}
}

func TestDispatchBareM(t *testing.T) {
	testkit.CleanEnv(t)
	projDir := setupDispatchFixture(t, "suggestions")
	cliRoot := cli.NewMRoot(cli.BuildInfo{Version: "0.0.0-test"})
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	cliRoot.SetOut(outBuf)
	cliRoot.SetErr(errBuf)
	cliRoot.SetArgs([]string{"--cwd", projDir})
	code := cli.ExecuteWithContext(cliRoot, context.Background())
	out := errBuf.String()
	if code != 2 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	if !strings.Contains(out, "ERR_M_USAGE") || !strings.Contains(out, "dev") {
		t.Fatalf("out=%s", out)
	}
}

func TestDispatchMalformedManifest(t *testing.T) {
	testkit.CleanEnv(t)
	testkit.EnableDirectScripts(t)
	projDir := t.TempDir()
	testkit.CopyFixture(t, "dispatch/malformed-manifest", projDir)
	code, out := runMProject(t, projDir, "dev")
	if code != 1 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	if !strings.Contains(out, "ERR_M_MANIFEST") {
		t.Fatalf("out=%s", out)
	}
}

func TestDispatchWorkspaceDirectRequiresBothGates(t *testing.T) {
	skipWithoutNode(t)
	testkit.CleanEnv(t)
	testkit.EnableDirectScripts(t)
	projDir := t.TempDir()
	testkit.CopyFixture(t, "workspace-runner/dag-simple", projDir)
	code, out := runMProject(t, projDir, "-r", "build")
	if code != 2 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	if !strings.Contains(out, "workspaces not enabled") {
		t.Fatalf("out=%s", out)
	}
}

func TestDispatchWorkspaceDirectWithGates(t *testing.T) {
	skipWithoutNode(t)
	projDir := setupDispatchWorkspaceFixture(t, "dag-simple")
	code, _ := runMProjectArgv(t, projDir, "-r", "build")
	if code != 0 {
		t.Fatalf("exit=%d want 0 for m -r build", code)
	}
}

func TestDispatchJSONBuiltin(t *testing.T) {
	testkit.CleanEnv(t)
	modRoot := testkit.ModuleRoot(t)
	code, out := runMProject(t, modRoot, "__dispatch", "install")
	if code != 0 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &doc); err != nil {
		t.Fatal(err)
	}
	if doc["kind"] != "builtin" {
		t.Fatalf("%v", doc)
	}
}

func TestExecuteMXUnchanged(t *testing.T) {
	testkit.CleanEnv(t)
	root := cli.NewMXRoot(cli.BuildInfo{Version: "0.0.0-test"})
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"version"})
	if code := cli.ExecuteWithContext(root, context.Background()); code != 0 {
		t.Fatalf("exit=%d", code)
	}
	if !strings.Contains(buf.String(), "mx") {
		t.Fatalf("out=%s", buf.String())
	}
}

func TestDispatchInstalTypo(t *testing.T) {
	testkit.CleanEnv(t)
	testkit.EnableDirectScripts(t)
	code, out := runMProject(t, t.TempDir(), "instal")
	if code != 2 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	if !strings.Contains(out, "m install") {
		t.Fatalf("out=%s", out)
	}
}
