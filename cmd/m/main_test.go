package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestFeaturesJSON(t *testing.T) {
	root := repoRoot(t)
	cmd := exec.Command("go", "run", "./cmd/m", "features", "--format=json")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			t.Fatalf("m features: %s\n%s", err, ee.Stderr)
		}
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `"id"`) || !strings.Contains(string(out), `"primary_mvp"`) {
		t.Fatalf("unexpected JSON output: %s", out)
	}
	if strings.Contains(string(out), `"tests"`) {
		t.Fatal("user-facing JSON must not include tests field")
	}
}

func TestFeaturesTableFilter(t *testing.T) {
	root := repoRoot(t)
	cmd := exec.Command("go", "run", "./cmd/m", "features", "--module", "runner", "--status", "planned")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	if !strings.Contains(text, "runner.direct-shortcuts") {
		t.Fatalf("expected runner features in output:\n%s", text)
	}
	if strings.Contains(text, "foundation.charter") {
		t.Fatal("filtered output must not include foundation rows")
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}
