package integration_test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

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
	// Use an absolute path so output lands in proj regardless of CWD.
	// Forward slashes keep the JSON string literal well-formed on every OS.
	outPath := filepath.ToSlash(filepath.Join(proj, "output.txt"))
	pkg := map[string]any{
		"scripts": map[string]string{
			"hello": "node -e \"require('fs').writeFileSync('" + outPath + "','script-wins\\n')\"",
		},
	}
	data, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(proj, "package.json"), string(data))
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
