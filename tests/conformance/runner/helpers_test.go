package runner_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/cli"
	"github.com/mewisme/mew/internal/registry"
	"github.com/mewisme/mew/internal/testkit"
)

func moduleRoot(t *testing.T) string {
	t.Helper()
	return testkit.ModuleRoot(t)
}

func skipWithoutNode(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node required")
	}
}

func setupRunnerFixture(t *testing.T, rel string) string {
	t.Helper()
	testkit.CleanEnv(t)
	projDir := t.TempDir()
	testkit.CopyFixture(t, "runner/"+rel, projDir)
	return projDir
}

func runMProject(t *testing.T, projDir string, args ...string) (int, string) {
	t.Helper()
	cliRoot := cli.NewMRoot(cli.BuildInfo{Version: "0.0.0-test"})
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	cliRoot.SetOut(outBuf)
	cliRoot.SetErr(errBuf)
	full := append([]string{"--cwd", projDir, "--reporter", "silent"}, args...)
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

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

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
	registry.TestProbeReset()
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

func fixturePath(t *testing.T, parts ...string) string {
	t.Helper()
	return filepath.Join(append([]string{moduleRoot(t), "fixtures"}, parts...)...)
}

func isWindows() bool { return runtime.GOOS == "windows" }
func isUnix() bool    { return runtime.GOOS != "windows" }

func parseJSON(t *testing.T, data string, v any) {
	t.Helper()
	if err := json.Unmarshal([]byte(strings.TrimSpace(data)), v); err != nil {
		t.Fatalf("json: %v data=%s", err, data)
	}
}
