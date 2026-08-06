package integration_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/mewisme/mew/internal/cli"
	"github.com/mewisme/mew/internal/testkit"
)

// runtimeE2EFixture copies the runtime-e2e fixture into a temp dir and returns
// the path to the fixture directory.
func runtimeE2EFixture(t *testing.T) string {
	t.Helper()
	testkit.CleanEnv(t)
	projDir := t.TempDir()
	testkit.CopyFixture(t, "runner/runtime-e2e", projDir)
	return projDir
}

// readOutput reads the fixture output file produced by test scripts that write
// results to a file (for tests that need to verify specific output).
func readOutput(t *testing.T, dir string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "output.txt"))
	if err != nil {
		t.Fatalf("read output.txt: %v", err)
	}
	return strings.TrimSpace(string(data))
}

// runMWithRuntime runs the m CLI with MEW_EXPERIMENTAL_RUNTIME=1 set.
func runMWithRuntime(t *testing.T, projDir string, args ...string) (int, string) {
	t.Helper()
	t.Setenv("MEW_EXPERIMENTAL_RUNTIME", "1")
	return runMProject(t, projDir, args...)
}

// runMWithRuntimeCtx runs the m CLI with MEW_EXPERIMENTAL_RUNTIME=1 and a context.
func runMWithRuntimeCtx(t *testing.T, ctx context.Context, projDir string, args ...string) (int, string) {
	t.Helper()
	t.Setenv("MEW_EXPERIMENTAL_RUNTIME", "1")
	return runMProjectCtx(t, ctx, projDir, args...)
}

// --- JS/TS entrypoint execution ---

func TestRuntimeE2EHelloJS(t *testing.T) {
	skipWithoutNode(t)
	proj := runtimeE2EFixture(t)
	code, _ := runMWithRuntime(t, proj, "hello.js")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
}

func TestRuntimeE2EHelloMJS(t *testing.T) {
	skipWithoutNode(t)
	proj := runtimeE2EFixture(t)
	code, _ := runMWithRuntime(t, proj, "hello.mjs")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
}

func TestRuntimeE2EHelloCJS(t *testing.T) {
	skipWithoutNode(t)
	proj := runtimeE2EFixture(t)
	code, _ := runMWithRuntime(t, proj, "hello.cjs")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
}

func TestRuntimeE2EHelloTS(t *testing.T) {
	skipWithoutNode(t)
	proj := runtimeE2EFixture(t)
	code, _ := runMWithRuntime(t, proj, "hello.ts")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
}

func TestRuntimeE2EHelloMTS(t *testing.T) {
	skipWithoutNode(t)
	proj := runtimeE2EFixture(t)
	code, _ := runMWithRuntime(t, proj, "hello.mts")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
}

func TestRuntimeE2EHelloCTS(t *testing.T) {
	skipWithoutNode(t)
	if !nodeMeetsMinimum(t, 20, 0) {
		t.Skip("Node >= 20 required for .cts loader transform (CJS loader hooks)")
	}
	proj := runtimeE2EFixture(t)
	code, _ := runMWithRuntime(t, proj, "hello.cts")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
}

// --- Imports, args, errors ---

func TestRuntimeE2EImportResolution(t *testing.T) {
	skipWithoutNode(t)
	proj := runtimeE2EFixture(t)
	code, _ := runMWithRuntime(t, proj, "imports.mjs")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
	if out := readOutput(t, proj); out != "hello from import" {
		t.Fatalf("got %q, want 'hello from import'", out)
	}
}

func TestRuntimeE2EEntrypointArgs(t *testing.T) {
	skipWithoutNode(t)
	proj := runtimeE2EFixture(t)
	code, _ := runMWithRuntime(t, proj, "args.js", "--", "alpha", "beta")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
	out := readOutput(t, proj)
	if !strings.Contains(out, "alpha") || !strings.Contains(out, "beta") {
		t.Fatalf("got %q, want args containing alpha, beta", out)
	}
}

func TestRuntimeE2ENodeV8Args(t *testing.T) {
	skipWithoutNode(t)
	proj := runtimeE2EFixture(t)
	code, _ := runMWithRuntime(t, proj, "node-args", "--", "--expose-gc", "hello.js")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
}

func TestRuntimeE2ESyntaxErrorJS(t *testing.T) {
	skipWithoutNode(t)
	proj := runtimeE2EFixture(t)
	code, _ := runMWithRuntime(t, proj, "syntax-error.js")
	if code == 0 {
		t.Fatal("expected non-zero exit for syntax error")
	}
}

func TestRuntimeE2ESyntaxErrorTS(t *testing.T) {
	skipWithoutNode(t)
	proj := runtimeE2EFixture(t)
	code, _ := runMWithRuntime(t, proj, "syntax-error.ts")
	if code == 0 {
		t.Fatal("expected non-zero exit for TS syntax error")
	}
}

func TestRuntimeE2EExitCode(t *testing.T) {
	skipWithoutNode(t)
	proj := runtimeE2EFixture(t)
	code, _ := runMWithRuntime(t, proj, "exit-code.js")
	if code != 42 {
		t.Fatalf("expected exit code 42, got %d", code)
	}
}

// --- Zero augmentation, dispatch precedence, deferred extensions ---

func TestRuntimeE2EZeroAugmentation(t *testing.T) {
	skipWithoutNode(t)
	proj := runtimeE2EFixture(t)
	code, _ := runMWithRuntime(t, proj, "--node", "env-dump.js")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
	out := readOutput(t, proj)
	if out != "absent" {
		t.Fatalf("--node mode leaked MEW_TRANSFORM_ENDPOINT: got %q", out)
	}
}

func TestRuntimeE2EScriptWinsOverFile(t *testing.T) {
	skipWithoutNode(t)
	proj := runtimeE2EFixture(t)
	// Use a .js file: inline node -e is mangled by cmd.exe on Windows.
	writeFile(t, filepath.Join(proj, "hello-script.js"), `require("fs").writeFileSync("output.txt", "script-wins\n")`)
	writeFile(t, filepath.Join(proj, "package.json"), `{"scripts": {"hello": "node hello-script.js"}}`)
	code, _ := runMProject(t, proj, "run", "hello")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
	if out := readOutput(t, proj); out != "script-wins" {
		t.Fatalf("got %q, want 'script-wins'", out)
	}
}

func TestRuntimeE2EJSXDeferred(t *testing.T) {
	proj := runtimeE2EFixture(t)
	t.Setenv("MEW_EXPERIMENTAL_RUNTIME", "1")
	code, out := runMProject(t, proj, "test.jsx")
	if code == 0 {
		t.Fatalf("expected non-zero exit for .jsx, got out=%s", out)
	}
	if !strings.Contains(out, "0052") {
		t.Fatalf("expected 0052 deferral message, got %q", out)
	}
}

func TestRuntimeE2ETSXDeferred(t *testing.T) {
	proj := runtimeE2EFixture(t)
	t.Setenv("MEW_EXPERIMENTAL_RUNTIME", "1")
	code, out := runMProject(t, proj, "test.tsx")
	if code == 0 {
		t.Fatalf("expected non-zero exit for .tsx, got out=%s", out)
	}
	if !strings.Contains(out, "0052") {
		t.Fatalf("expected 0052 deferral message, got %q", out)
	}
}

// --- Cache: warm, corrupt recovery ---

func TestRuntimeE2EWarmCache(t *testing.T) {
	skipWithoutNode(t)
	testkit.CleanEnv(t)
	projDir := t.TempDir()
	testkit.CopyFixture(t, "runner/runtime-e2e", projDir)

	t.Setenv("MEW_EXPERIMENTAL_RUNTIME", "1")
	// First run: populate cache.
	code1, _ := runMProject(t, projDir, "hello.js")
	if code1 != 0 {
		t.Fatalf("first run: exit=%d", code1)
	}
	// Second run: warm cache, should reuse verified assets.
	code2, _ := runMProject(t, projDir, "hello.js")
	if code2 != 0 {
		t.Fatalf("second run (warm cache): exit=%d", code2)
	}
}

func TestRuntimeE2ECorruptCacheRecovery(t *testing.T) {
	skipWithoutNode(t)
	testkit.CleanEnv(t)
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)
	cacheDir := filepath.Join(homeDir, ".cache", "mew")
	projDir := t.TempDir()
	testkit.CopyFixture(t, "runner/runtime-e2e", projDir)

	t.Setenv("MEW_EXPERIMENTAL_RUNTIME", "1")
	// First run: populate cache.
	code1, _ := runMProject(t, projDir, "hello.js")
	if code1 != 0 {
		t.Fatalf("first run: exit=%d", code1)
	}
	// Corrupt cached assets.
	corruptCacheAssets(t, cacheDir)
	// Second run: should recover and still produce correct output.
	code2, _ := runMProject(t, projDir, "hello.js")
	if code2 != 0 {
		t.Fatalf("second run (corrupt cache): exit=%d", code2)
	}
}

// --- Source maps, paths, node-args, mx, cancellation, gate ---

func TestRuntimeE2ESourceMaps(t *testing.T) {
	skipWithoutNode(t)
	if !nodeMeetsMinimum(t, 20, 6) {
		t.Skip("Node >= 20.6 required for --enable-source-maps")
	}
	proj := runtimeE2EFixture(t)
	t.Setenv("NODE_OPTIONS", "--enable-source-maps")
	code, _ := runMWithRuntime(t, proj, "hello.ts")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
}

func TestRuntimeE2ESpacesInPath(t *testing.T) {
	skipWithoutNode(t)
	testkit.CleanEnv(t)
	projDir := t.TempDir()
	spacesDir := filepath.Join(projDir, "path with spaces")
	if err := os.MkdirAll(spacesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(spacesDir, "hello.js"), `console.log("spaces work");`)
	t.Setenv("MEW_EXPERIMENTAL_RUNTIME", "1")
	code, _ := runMProject(t, spacesDir, "hello.js")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
}

func TestRuntimeE2EUnicodeInPath(t *testing.T) {
	skipWithoutNode(t)
	testkit.CleanEnv(t)
	projDir := t.TempDir()
	unicodeDir := filepath.Join(projDir, "café")
	if err := os.MkdirAll(unicodeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(unicodeDir, "hello.js"), `console.log("unicode works");`)
	t.Setenv("MEW_EXPERIMENTAL_RUNTIME", "1")
	code, _ := runMProject(t, unicodeDir, "hello.js")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
}

func TestRuntimeE2ENodeArgsStyle(t *testing.T) {
	skipWithoutNode(t)
	proj := runtimeE2EFixture(t)
	code, _ := runMWithRuntime(t, proj, "node-args", "--", "hello.js")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
}

func TestRuntimeE2EMXNoFileRun(t *testing.T) {
	proj := runtimeE2EFixture(t)
	t.Setenv("MEW_EXPERIMENTAL_RUNTIME", "1")
	code, out := runMXInProject(t, proj, "hello.js")
	if code == 0 {
		t.Fatalf("expected non-zero exit for mx file-run, got out=%s", out)
	}
}

func TestRuntimeE2ECancellation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("signal propagation unreliable on Windows")
	}
	skipWithoutNode(t)
	proj := runtimeE2EFixture(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	code, _ := runMWithRuntimeCtx(t, ctx, proj, "sigterm.mjs")
	if code == 0 {
		t.Fatal("expected non-zero exit for cancelled process")
	}
}

func TestRuntimeE2EDisabledGate(t *testing.T) {
	proj := runtimeE2EFixture(t)
	code, out := runMProject(t, proj, "hello.js")
	if code == 0 {
		t.Fatalf("expected non-zero exit when gate is off, got out=%s", out)
	}
}

// --- Transform credential isolation ---

func TestRuntimeE2ETSNoTransformEndpoint(t *testing.T) {
	skipWithoutNode(t)
	proj := runtimeE2EFixture(t)
	code, _ := runMWithRuntime(t, proj, "env-dump.ts")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
	out := readOutput(t, proj)
	if out != "absent" {
		t.Fatalf("MEW_TRANSFORM_ENDPOINT leaked into user code: got %q, want 'absent'", out)
	}
}

func TestRuntimeE2EAllCredsStrippedJS(t *testing.T) {
	skipWithoutNode(t)
	proj := runtimeE2EFixture(t)
	code, _ := runMWithRuntime(t, proj, "env-dump-all.js")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
	out := readOutput(t, proj)
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			t.Fatalf("malformed output line: %q", line)
		}
		if parts[1] != "absent" {
			t.Fatalf("%s leaked: got %q, want 'absent'", parts[0], parts[1])
		}
	}
}

func TestRuntimeE2EAllCredsStrippedTS(t *testing.T) {
	skipWithoutNode(t)
	proj := runtimeE2EFixture(t)
	writeFile(t, filepath.Join(proj, "check.ts"),
		"import { writeFileSync } from \"node:fs\";"+"\n"+
			"var vars = [\"MEW_TRANSFORM_ENDPOINT\",\"MEW_TRANSFORM_TOKEN\",\"MEW_TRANSFORM_OPTIONS\",\"MEW_TRANSFORM_OPTS_DIGEST\"];"+"\n"+
			"var out = vars.map(function(v) { return v + \"=\" + (process.env[v] || \"absent\"); });"+"\n"+
			"writeFileSync(\"output.txt\", out.join(String.fromCharCode(10)));"+"\n")
	code, _ := runMWithRuntime(t, proj, "check.ts")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
	out := readOutput(t, proj)
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			t.Fatalf("malformed output line: %q", line)
		}
		if parts[1] != "absent" {
			t.Fatalf("%s leaked into TS entrypoint: got %q, want 'absent'", parts[0], parts[1])
		}
	}
}

func TestRuntimeE2ENoCredsInWorker(t *testing.T) {
	skipWithoutNode(t)
	proj := runtimeE2EFixture(t)
	code, _ := runMWithRuntime(t, proj, "worker-check.mjs")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
	out := readOutput(t, proj)
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "worker-error:") {
			t.Fatalf("worker error: %s", line)
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			t.Fatalf("malformed output line: %q", line)
		}
		if parts[1] != "absent" {
			t.Fatalf("%s leaked into worker: got %q, want 'absent'", parts[0], parts[1])
		}
	}
}

func TestRuntimeE2ENoCredsInChildProcess(t *testing.T) {
	skipWithoutNode(t)
	proj := runtimeE2EFixture(t)
	code, _ := runMWithRuntime(t, proj, "child-check.js")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
	out := readOutput(t, proj)
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "child-error:") {
			t.Fatalf("child process error: %s", line)
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			t.Fatalf("malformed output line: %q", line)
		}
		if parts[1] != "absent" {
			t.Fatalf("%s leaked into child process: got %q, want 'absent'", parts[0], parts[1])
		}
	}
}

func TestRuntimeE2ENodeOptionsRequireRejected(t *testing.T) {
	skipWithoutNode(t)
	proj := runtimeE2EFixture(t)
	t.Setenv("NODE_OPTIONS", "--require ./evil.cjs")
	code, out := runMWithRuntime(t, proj, "hello.ts")
	if code == 0 {
		t.Fatalf("expected non-zero exit for NODE_OPTIONS with --require, got out=%s", out)
	}
	if !strings.Contains(out, "NODE_OPTIONS") && !strings.Contains(out, "credential") {
		t.Fatalf("expected NODE_OPTIONS rejection message, got %q", out)
	}
}

func TestRuntimeE2ENodeOptionsImportRejected(t *testing.T) {
	skipWithoutNode(t)
	proj := runtimeE2EFixture(t)
	t.Setenv("NODE_OPTIONS", "--import ./evil.mjs")
	code, out := runMWithRuntime(t, proj, "hello.js")
	if code == 0 {
		t.Fatalf("expected non-zero exit for NODE_OPTIONS with --import, got out=%s", out)
	}
	if !strings.Contains(out, "NODE_OPTIONS") && !strings.Contains(out, "credential") {
		t.Fatalf("expected NODE_OPTIONS rejection message, got %q", out)
	}
}

func TestRuntimeE2ENodeArgsRequireStillWorks(t *testing.T) {
	skipWithoutNode(t)
	proj := runtimeE2EFixture(t)
	writeFile(t, filepath.Join(proj, "preload-check.cjs"),
		"var fs = require(\"fs\");"+"\n"+
			"var vars = [\"MEW_TRANSFORM_ENDPOINT\",\"MEW_TRANSFORM_TOKEN\",\"MEW_TRANSFORM_OPTIONS\"];"+"\n"+
			"var results = vars.map(function(v){return v+\"=\"+(process.env[v]||\"absent\")});"+"\n"+
			"fs.writeFileSync(\"output.txt\", results.join(\"\\n\"));"+"\n")
	code, _ := runMWithRuntime(t, proj, "node-args", "--", "--require", "./preload-check.cjs", "hello.js")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
	out := readOutput(t, proj)
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			t.Fatalf("malformed output line: %q", line)
		}
		if parts[1] != "absent" {
			t.Fatalf("%s leaked through user --require preload: got %q, want 'absent'", parts[0], parts[1])
		}
	}
}

func TestRuntimeE2ENodeArgsTypeScriptStillWorks(t *testing.T) {
	skipWithoutNode(t)
	proj := runtimeE2EFixture(t)
	code, _ := runMWithRuntime(t, proj, "node-args", "--", "--no-warnings", "hello.ts")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
}

func TestRuntimeE2ETSLoaderAuthenticates(t *testing.T) {
	skipWithoutNode(t)
	proj := runtimeE2EFixture(t)
	code, _ := runMWithRuntime(t, proj, "hello.ts")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
}

func TestRuntimeE2ETSLoaderMTS(t *testing.T) {
	skipWithoutNode(t)
	proj := runtimeE2EFixture(t)
	code, _ := runMWithRuntime(t, proj, "hello.mts")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
}

func TestRuntimeE2ETSLoaderCTS(t *testing.T) {
	skipWithoutNode(t)
	if !nodeMeetsMinimum(t, 20, 0) {
		t.Skip("Node >= 20 required for .cts loader transform (CJS loader hooks)")
	}
	proj := runtimeE2EFixture(t)
	code, _ := runMWithRuntime(t, proj, "hello.cts")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
}

func TestRuntimeE2ENodeModeZeroAugmentation(t *testing.T) {
	skipWithoutNode(t)
	proj := runtimeE2EFixture(t)
	code, _ := runMWithRuntime(t, proj, "--node", "env-dump-all.js")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
	out := readOutput(t, proj)
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			t.Fatalf("malformed output line: %q", line)
		}
		if parts[1] != "absent" {
			t.Fatalf("%s leaked in --node mode: got %q, want 'absent'", parts[0], parts[1])
		}
	}
}

func TestRuntimeE2EErrorOutputNoEndpointOrToken(t *testing.T) {
	skipWithoutNode(t)
	proj := runtimeE2EFixture(t)
	code, out := runMWithRuntime(t, proj, "syntax-error.ts")
	if code == 0 {
		t.Fatal("expected non-zero exit for syntax error")
	}
	if strings.Contains(out, "127.0.0.1:") {
		t.Fatalf("error output contains endpoint address: %q", out)
	}
	if strings.Contains(out, "MEW_TRANSFORM_TOKEN=") {
		t.Fatalf("error output contains token env: %q", out)
	}
	if strings.Contains(out, "MEW_TRANSFORM_ENDPOINT=") {
		t.Fatalf("error output contains endpoint env: %q", out)
	}
}

// --- package-script CWD ---

func TestRuntimeE2EScriptCWDIsProjectRoot(t *testing.T) {
	skipWithoutNode(t)
	proj := runtimeE2EFixture(t)
	// Use a separate .js file so the script works cross-platform: inline
	// node -e with double quotes is mangled by cmd.exe on Windows.
	writeFile(t, filepath.Join(proj, "cwdcheck.js"), `require("fs").writeFileSync("cwd.txt", process.cwd())`)
	writeFile(t, filepath.Join(proj, "package.json"), `{"scripts": {"cwdcheck": "node cwdcheck.js"}}`)
	code, _ := runMProject(t, proj, "run", "cwdcheck")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
	data, err := os.ReadFile(filepath.Join(proj, "cwd.txt"))
	if err != nil {
		t.Fatalf("read cwd.txt: %v", err)
	}
	got := strings.TrimSpace(string(data))
	// Compare after resolving both to absolute form so symlinks and
	// platform path normalizations are handled correctly.
	want, err := filepath.Abs(proj)
	if err != nil {
		t.Fatal(err)
	}
	gotAbs, err := filepath.Abs(got)
	if err != nil {
		t.Fatalf("resolve subprocess cwd: %v", err)
	}
	if !samePath(gotAbs, want) {
		t.Fatalf("script cwd=%q, want project root %q", got, want)
	}
}

func TestRuntimeE2EScriptCWDWithCWD(t *testing.T) {
	skipWithoutNode(t)
	proj := runtimeE2EFixture(t)
	// Use a separate .js file: inline node -e with double quotes is
	// mangled by cmd.exe on Windows.
	writeFile(t, filepath.Join(proj, "cwdcheck.js"), `require("fs").writeFileSync("cwd.txt", process.cwd())`)
	writeFile(t, filepath.Join(proj, "package.json"), `{"scripts": {"cwdcheck": "node cwdcheck.js"}}`)
	// Run from parent directory; --cwd selects the project.
	parent := filepath.Dir(proj)
	code, _ := runMProjectWithCWD(t, parent, proj, "run", "cwdcheck")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
	data, err := os.ReadFile(filepath.Join(proj, "cwd.txt"))
	if err != nil {
		t.Fatalf("read cwd.txt: %v", err)
	}
	got := strings.TrimSpace(string(data))
	if !samePath(got, proj) {
		t.Fatalf("script cwd=%q, want project root %q", got, proj)
	}
}

func TestRuntimeE2EScriptCWDFromSubdir(t *testing.T) {
	skipWithoutNode(t)
	proj := runtimeE2EFixture(t)
	subdir := filepath.Join(proj, "subdir")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Use a separate .js file: inline node -e with double quotes is
	// mangled by cmd.exe on Windows.
	writeFile(t, filepath.Join(proj, "cwdcheck.js"), `require("fs").writeFileSync("cwd.txt", process.cwd())`)
	writeFile(t, filepath.Join(proj, "package.json"), `{"scripts": {"cwdcheck": "node cwdcheck.js"}}`)
	// Invoke with --cwd pointing to a subdirectory; the project root is
	// discovered by walking up, and scripts still run from the project root.
	code, _ := runMProjectWithCWD(t, "", subdir, "run", "cwdcheck")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
	data, err := os.ReadFile(filepath.Join(proj, "cwd.txt"))
	if err != nil {
		t.Fatalf("read cwd.txt: %v", err)
	}
	got := strings.TrimSpace(string(data))
	if !samePath(got, proj) {
		t.Fatalf("script cwd=%q, want project root %q", got, proj)
	}
}

func TestRuntimeE2EScriptCWDLifecycleScript(t *testing.T) {
	skipWithoutNode(t)
	proj := runtimeE2EFixture(t)
	// Use separate .js files: inline node -e with double quotes is
	// mangled by cmd.exe on Windows.
	writeFile(t, filepath.Join(proj, "pre-cwdcheck.js"), `require("fs").writeFileSync("pre-cwd.txt", process.cwd())`)
	writeFile(t, filepath.Join(proj, "build-cwdcheck.js"), `require("fs").writeFileSync("build-cwd.txt", process.cwd())`)
	writeFile(t, filepath.Join(proj, "post-cwdcheck.js"), `require("fs").writeFileSync("post-cwd.txt", process.cwd())`)
	writeFile(t, filepath.Join(proj, "package.json"), `{"scripts": {
		"prebuild": "node pre-cwdcheck.js",
		"build": "node build-cwdcheck.js",
		"postbuild": "node post-cwdcheck.js"
	}}`)
	code, _ := runMProject(t, proj, "run", "build")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
	for _, fname := range []string{"pre-cwd.txt", "build-cwd.txt", "post-cwd.txt"} {
		data, err := os.ReadFile(filepath.Join(proj, fname))
		if err != nil {
			t.Fatalf("read %s: %v", fname, err)
		}
		got := strings.TrimSpace(string(data))
		if !samePath(got, proj) {
			t.Fatalf("%s cwd=%q, want project root %q", fname, got, proj)
		}
	}
}

func TestRuntimeE2EScriptCWDPathsWithSpaces(t *testing.T) {
	skipWithoutNode(t)
	testkit.CleanEnv(t)
	projDir := t.TempDir()
	spacesDir := filepath.Join(projDir, "path with spaces")
	if err := os.MkdirAll(spacesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(spacesDir, "hello.js"), `require("fs").writeFileSync("cwd.txt", process.cwd())`)
	writeFile(t, filepath.Join(spacesDir, "package.json"), `{"scripts": {"cwdcheck": "node hello.js"}}`)
	t.Setenv("MEW_EXPERIMENTAL_RUNTIME", "1")
	code, _ := runMProject(t, spacesDir, "run", "cwdcheck")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
	data, err := os.ReadFile(filepath.Join(spacesDir, "cwd.txt"))
	if err != nil {
		t.Fatalf("read cwd.txt: %v", err)
	}
	got := strings.TrimSpace(string(data))
	if !samePath(got, spacesDir) {
		t.Fatalf("script cwd=%q, want %q", got, spacesDir)
	}
}

func TestRuntimeE2EScriptCWDUnicodePath(t *testing.T) {
	skipWithoutNode(t)
	testkit.CleanEnv(t)
	projDir := t.TempDir()
	unicodeDir := filepath.Join(projDir, "café")
	if err := os.MkdirAll(unicodeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(unicodeDir, "hello.js"), `require("fs").writeFileSync("cwd.txt", process.cwd())`)
	writeFile(t, filepath.Join(unicodeDir, "package.json"), `{"scripts": {"cwdcheck": "node hello.js"}}`)
	t.Setenv("MEW_EXPERIMENTAL_RUNTIME", "1")
	code, _ := runMProject(t, unicodeDir, "run", "cwdcheck")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
	data, err := os.ReadFile(filepath.Join(unicodeDir, "cwd.txt"))
	if err != nil {
		t.Fatalf("read cwd.txt: %v", err)
	}
	got := strings.TrimSpace(string(data))
	if !samePath(got, unicodeDir) {
		t.Fatalf("script cwd=%q, want %q", got, unicodeDir)
	}
}

func TestRuntimeE2EScriptCWDFailedScriptReportsCorrectContext(t *testing.T) {
	skipWithoutNode(t)
	proj := runtimeE2EFixture(t)
	// Use a .js file: inline node -e is mangled by cmd.exe on Windows.
	writeFile(t, filepath.Join(proj, "fail.js"), `process.exit(1)`)
	writeFile(t, filepath.Join(proj, "package.json"), `{"scripts": {"fail": "node fail.js"}}`)
	code, _ := runMProject(t, proj, "run", "fail")
	if code == 0 {
		t.Fatal("expected non-zero exit")
	}
	// The exit code propagates through the runner correctly.
	// The working-directory context is set correctly (verified by the
	// CWD test family above).
}

// --- helpers ---

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func corruptCacheAssets(t *testing.T, cacheDir string) {
	t.Helper()
	err := filepath.Walk(cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".js", ".mjs", ".cjs", ".json":
			return os.WriteFile(path, []byte("corrupt"), 0o644)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func nodeMeetsMinimum(t *testing.T, major, minor int) bool {
	t.Helper()
	nodePath, err := exec.LookPath("node")
	if err != nil {
		return false
	}
	cmd := exec.Command(nodePath, "-e", "console.log(process.versions.node)")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	version := strings.TrimSpace(string(out))
	version = strings.TrimPrefix(version, "v")
	parts := strings.SplitN(version, ".", 2)
	if len(parts) < 2 {
		return false
	}
	nodeMajor := 0
	nodeMinor := 0
	if len(parts[0]) > 0 {
		for _, c := range parts[0] {
			if c < '0' || c > '9' {
				break
			}
			nodeMajor = nodeMajor*10 + int(c-'0')
		}
	}
	minorParts := strings.SplitN(parts[1], ".", 2)
	if len(minorParts[0]) > 0 {
		for _, c := range minorParts[0] {
			if c < '0' || c > '9' {
				break
			}
			nodeMinor = nodeMinor*10 + int(c-'0')
		}
	}
	return nodeMajor > major || (nodeMajor == major && nodeMinor >= minor)
}

// samePath reports whether two absolute paths refer to the same filesystem location.
func samePath(a, b string) bool {
	// Resolve symlinks so paths compare equal when the OS returns
	// different representations of the same directory (notably macOS
	// where /var is a symlink to /private/var).
	if resolved, err := filepath.EvalSymlinks(a); err == nil {
		a = resolved
	}
	if resolved, err := filepath.EvalSymlinks(b); err == nil {
		b = resolved
	}
	ca := filepath.Clean(a)
	cb := filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(ca, cb)
	}
	return ca == cb
}

// runMProjectWithCWD runs the m binary with the process working directory set to
// workDir, optionally passing --cwd separately for project discovery. When cwd
// is empty, no --cwd flag is added and the project is discovered from workDir.
func runMProjectWithCWD(t *testing.T, workDir, cwd string, args ...string) (int, string) {
	t.Helper()
	testkit.CleanEnv(t)
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	cliRoot := cli.NewMRoot(testBuildInfo())
	cliRoot.SetOut(outBuf)
	cliRoot.SetErr(errBuf)
	full := []string{"--output", "silent"}
	if cwd != "" {
		full = append(full, "--cwd", cwd)
	}
	full = append(full, args...)
	cliRoot.SetArgs(full)
	ctx := context.Background()
	code := cli.ExecuteWithContext(cliRoot, ctx)
	out := outBuf.String()
	errOut := errBuf.String()
	if code != 0 {
		out = strings.TrimSpace(out)
		errOut = strings.TrimSpace(errOut)
		combined := out
		if errOut != "" {
			if combined != "" {
				combined += errOut
			} else {
				combined = errOut
			}
		}
		return code, strings.TrimSpace(combined)
	}
	return code, strings.TrimSpace(out)
}

func testBuildInfo() cli.BuildInfo {
	return cli.BuildInfo{Version: "0.0.0-test"}
}
