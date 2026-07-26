package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAllowlistRejectsUnknown(t *testing.T) {
	root := findRoot(t)
	allowPath := filepath.Join(root, "tools", "allowlist", "modules.txt")
	b, err := os.ReadFile(allowPath)
	if err != nil {
		t.Fatal(err)
	}
	allow := map[string]bool{}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		allow[line] = true
	}
	fake := "github.com/example/banned-module"
	if allow[fake] {
		t.Fatalf("probe module %q must not be allowlisted", fake)
	}
}

func findRoot(t *testing.T) string {
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
