package runner_test

import (
	"runtime"
	"testing"
)

func TestLaunchNodeShebangUnix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-only suite")
	}
	skipWithoutNode(t)
	proj := setupRunnerFixture(t, "basic-scripts")
	code, _ := runMProject(t, proj, "run", "dev")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
}

func TestLaunchMetacharLiteralUnix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-only suite")
	}
	skipWithoutNode(t)
	proj := setupRunnerFixture(t, "shell-quoting")
	code, out := runMProject(t, proj, "run", "print", "--", ";rm -rf /")
	if code != 0 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
}
