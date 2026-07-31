package runner_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/runner"
	"github.com/mewisme/mew/internal/testkit"
	"github.com/mewisme/mew/internal/workspace"
)

func setupWorkspaceFixture(t *testing.T, rel string) string {
	t.Helper()
	testkit.EnableWorkspaces(t)
	testkit.CleanEnv(t)
	proj := t.TempDir()
	testkit.CopyFixture(t, "workspace-runner/"+rel, proj)
	return proj
}

func TestWorkspaceSchedulerReadyQueueOrder(t *testing.T) {
	root := setupWorkspaceFixture(t, "dag-simple")
	g, err := workspace.BuildGraph(root)
	if err != nil {
		t.Fatal(err)
	}
	paths, err := runner.SelectMembers(g, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	sched, err := runner.BuildSchedule(g, paths, runner.OrderTopological, 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(sched.Tasks) != 3 {
		t.Fatalf("tasks=%d", len(sched.Tasks))
	}
}

func TestWorkspaceSchedulerDependencyOrdering(t *testing.T) {
	root := setupWorkspaceFixture(t, "dag-simple")
	g, err := workspace.BuildGraph(root)
	if err != nil {
		t.Fatal(err)
	}
	paths, _ := runner.SelectMembers(g, true, nil)
	sched, err := runner.BuildSchedule(g, paths, runner.OrderTopological, 1)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]int{}
	for i, task := range sched.Tasks {
		seen[task.Path] = i
	}
	if seen["packages/lib"] <= seen["packages/base"] {
		t.Fatalf("lib should run after base: %v", seen)
	}
}

func TestWorkspaceBailStopsNewTasks(t *testing.T) {
	skipWithoutNode(t)
	proj := setupWorkspaceFixture(t, "failure-bail")
	code, _ := runMProject(t, proj, "-r", "run", "build", "--workspace-order", "sequential")
	if code != 7 {
		t.Fatalf("exit=%d want 7", code)
	}
	if _, err := os.Stat(filepath.Join(proj, "packages", "later", "out.txt")); err == nil {
		t.Fatal("later package should not run under bail")
	}
}

func TestWorkspaceBailTriggerExitHint(t *testing.T) {
	skipWithoutNode(t)
	proj := setupWorkspaceFixture(t, "failure-bail")
	code, _ := runMProject(t, proj, "-r", "run", "build", "--workspace-order", "sequential")
	if code != 7 {
		t.Fatalf("exit=%d", code)
	}
}

func TestWorkspaceOutputInheritMode(t *testing.T) {
	skipWithoutNode(t)
	proj := setupWorkspaceFixture(t, "large")
	code, _ := runMProject(t, proj, "-r", "run", "build", "--workspace-order", "parallel")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
}

func TestWorkspaceOutputPrefixMode(t *testing.T) {
	skipWithoutNode(t)
	proj := setupWorkspaceFixture(t, "large")
	code, _ := runMProject(t, proj, "-r", "run", "build", "--workspace-order", "parallel", "--workspace-output", "stream")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
}

func TestWorkspaceOutputJSONMode(t *testing.T) {
	skipWithoutNode(t)
	proj := setupWorkspaceFixture(t, "large")
	code, _ := runMProject(t, proj, "-r", "run", "build", "--workspace-order", "parallel", "--output", "json")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
}
