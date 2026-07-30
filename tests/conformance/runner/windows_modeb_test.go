package runner_test

import (
	"runtime"
	"testing"
)

func TestWindowsModeBDispatch(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-only suite")
	}
	skipWithoutNode(t)
	proj := setupRunnerFixture(t, "basic-scripts")
	code, _ := runMProject(t, proj, "run", "dev")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
}
