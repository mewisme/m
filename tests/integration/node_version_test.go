package integration_test

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/node"
	"github.com/mewisme/mew/internal/testkit"
)

func TestNodeVersionDetectCapabilities(t *testing.T) {
	skipWithoutNode(t)

	inst, err := node.Discover(context.Background(), node.Request{
		WorkingDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	t.Logf("Node: %s version=%s caps=%v", inst.ExePath, inst.NormalizedVersion, inst.Capabilities)

	// All supported Node versions must have require-preload (>=12) and import-preload (>=16).
	if !hasCap(inst.Capabilities, "require-preload") {
		t.Error("missing required capability: require-preload")
	}
	if !hasCap(inst.Capabilities, "import-preload") {
		t.Error("missing required capability: import-preload")
	}

	// TS requires module-register (>=18.19).
	if !hasCap(inst.Capabilities, "module-register") {
		t.Log("module-register not available (Node < 18.19)")
	}
}

func TestNodeVersionJSExecution(t *testing.T) {
	skipWithoutNode(t)

	// JS entrypoints work with Node >= 16 (import-preload).
	testkit.CleanEnv(t)
	proj := t.TempDir()
	writeFile(t, filepath.Join(proj, "app.js"), `console.log("ok");`)
	t.Setenv("MEW_EXPERIMENTAL_RUNTIME", "1")
	code, _ := runMProject(t, proj, "app.js")
	if code != 0 {
		t.Fatalf("JS execution failed: exit=%d", code)
	}
}

func TestNodeVersionTSExecution(t *testing.T) {
	skipWithoutNode(t)

	// TS entrypoints require module-register (Node >= 18.19).
	inst, err := node.Discover(context.Background(), node.Request{WorkingDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if !hasCap(inst.Capabilities, "module-register") {
		t.Skip("Node >= 18.19 required for TypeScript execution")
	}

	testkit.CleanEnv(t)
	proj := t.TempDir()
	writeFile(t, filepath.Join(proj, "app.ts"), `const x: string = "ts-ok"; console.log(x);`)
	t.Setenv("MEW_EXPERIMENTAL_RUNTIME", "1")
	code, _ := runMProject(t, proj, "app.ts")
	if code != 0 {
		t.Fatalf("TS execution failed: exit=%d", code)
	}
}

func TestNodeVersionZeroAugmentation(t *testing.T) {
	skipWithoutNode(t)

	// --node (zero augmentation) works on any Node >= 12.
	testkit.CleanEnv(t)
	proj := t.TempDir()
	writeFile(t, filepath.Join(proj, "app.js"), `console.log("stock");`)
	t.Setenv("MEW_EXPERIMENTAL_RUNTIME", "1")
	code, _ := runMProject(t, proj, "--node", "app.js")
	if code != 0 {
		t.Fatalf("--node execution failed: exit=%d", code)
	}
}

func TestNodeVersionExplicitCandidate(t *testing.T) {
	skipWithoutNode(t)

	nodePath, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node required")
	}

	inst, err := node.Discover(context.Background(), node.Request{
		WorkingDir:        t.TempDir(),
		ExplicitCandidate: nodePath,
	})
	if err != nil {
		t.Fatalf("Discover with explicit candidate: %v", err)
	}
	if inst.DiscoverySource != "explicit" {
		t.Errorf("discovery source: got %q, want 'explicit'", inst.DiscoverySource)
	}
	if inst.ExePath != nodePath {
		t.Errorf("exe path: got %q, want %q", inst.ExePath, nodePath)
	}
}

func TestNodeVersionNotFound(t *testing.T) {
	_, err := node.Discover(context.Background(), node.Request{
		WorkingDir:        t.TempDir(),
		ExplicitCandidate: "/nonexistent/node",
	})
	if err == nil {
		t.Fatal("expected error for nonexistent node")
	}
}

// --- helpers ---

func hasCap(caps []string, target string) bool {
	for _, c := range caps {
		if c == target {
			return true
		}
	}
	return false
}
