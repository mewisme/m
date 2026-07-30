package runner_test

import (
	"runtime"
	"testing"
)

func TestLaunchCmdShimWindows(t *testing.T) {
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

func TestLaunchComSpecWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-only suite")
	}
	skipWithoutNode(t)
	proj := setupRunnerFixture(t, "shell-quoting")
	code, _ := runMProject(t, proj, "run", "args", "--", "win")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
}

func TestLaunchSpacesWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-only suite")
	}
	skipWithoutNode(t)
	proj := setupRunnerFixture(t, "shell-quoting")
	code, _ := runMProject(t, proj, "run", "args", "--", "a b")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
}
