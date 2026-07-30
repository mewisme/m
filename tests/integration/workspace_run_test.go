package integration_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/testkit"
)

func setupWorkspaceRunFixture(t *testing.T, rel string) string {
	t.Helper()
	testkit.EnableWorkspaces(t)
	testkit.CleanEnv(t)
	projDir := t.TempDir()
	testkit.CopyFixture(t, "workspace-runner/"+rel, projDir)
	return projDir
}

func TestWorkspaceRunDAGSimple(t *testing.T) {
	skipWithoutNode(t)
	projDir := setupWorkspaceRunFixture(t, "dag-simple")

	code, out := runMProject(t, projDir, "-r", "run", "build")
	if code != 0 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	for _, name := range []string{"base", "lib", "app"} {
		marker := filepath.Join(projDir, ".results", name+".done")
		if _, err := os.Stat(marker); err != nil {
			t.Fatalf("missing %s: %v", marker, err)
		}
	}
}

func TestWorkspaceRunCycleFails(t *testing.T) {
	projDir := setupWorkspaceRunFixture(t, "cycle")
	code, out := runMProject(t, projDir, "-r", "run", "build")
	if code != 1 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	if !strings.Contains(out, "ERR_M_RESOLVE") && !strings.Contains(out, "cyclic") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestWorkspaceRunBailStopsLaterPackages(t *testing.T) {
	skipWithoutNode(t)
	projDir := setupWorkspaceRunFixture(t, "failure-bail")
	code, _ := runMProject(t, projDir, "-r", "run", "build", "--workspace-order", "sequential")
	if code != 7 {
		t.Fatalf("exit=%d want 7", code)
	}
	if _, err := os.Stat(filepath.Join(projDir, "packages", "later", "out.txt")); err == nil {
		t.Fatal("later package should not have run under bail")
	}
}

func TestWorkspaceRunContinueRunsAll(t *testing.T) {
	skipWithoutNode(t)
	projDir := setupWorkspaceRunFixture(t, "failure-bail")
	code, _ := runMProject(t, projDir, "-r", "run", "build", "--workspace-order", "parallel", "--no-workspace-bail")
	if code != 7 {
		t.Fatalf("exit=%d want 7", code)
	}
	if _, err := os.Stat(filepath.Join(projDir, "packages", "later", "out.txt")); err != nil {
		t.Fatal("later package should run in continue mode")
	}
}

func TestWorkspaceOnlyFlagsWithoutTrigger(t *testing.T) {
	projDir := setupWorkspaceRunFixture(t, "dag-simple")
	code, out := runMProject(t, projDir, "run", "build", "--workspace-concurrency", "2")
	if code != 2 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	if !strings.Contains(out, "ERR_M_USAGE") {
		t.Fatalf("out=%s", out)
	}
}

func TestWorkspaceRunPrefixAttribution(t *testing.T) {
	skipWithoutNode(t)
	projDir := setupWorkspaceRunFixture(t, "large")
	code, _ := runMProject(t, projDir, "-r", "run", "build", "--workspace-order", "parallel", "--workspace-concurrency", "4")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
	if _, err := os.Stat(filepath.Join(projDir, "packages", "pkg01", "out.txt")); err != nil {
		t.Fatal("pkg01 did not run")
	}
}
