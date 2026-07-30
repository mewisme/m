package envexec_test

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

func TestEnvexecMustNotImportAppOrCLI(t *testing.T) {
	t.Helper()
	cmd := exec.Command("go", "list", "-json", "github.com/mewisme/mew/internal/runner/envexec")
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			t.Fatalf("go list: %v\n%s", err, ee.Stderr)
		}
		t.Fatal(err)
	}
	var pkg struct {
		ImportPath string
		Imports    []string
	}
	if err := json.Unmarshal(out, &pkg); err != nil {
		t.Fatal(err)
	}
	forbidden := []string{
		"github.com/mewisme/mew/internal/app",
		"github.com/mewisme/mew/internal/cli",
	}
	for _, imp := range pkg.Imports {
		for _, bad := range forbidden {
			if imp == bad || strings.HasPrefix(imp, bad+"/") {
				t.Errorf("internal/runner/envexec must not import %s", imp)
			}
		}
	}
}

func TestRunnerMustNotImportEnvexec(t *testing.T) {
	t.Helper()
	cmd := exec.Command("go", "list", "-json", "github.com/mewisme/mew/internal/runner")
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			t.Fatalf("go list: %v\n%s", err, ee.Stderr)
		}
		t.Fatal(err)
	}
	var pkg struct {
		ImportPath string
		Imports    []string
	}
	if err := json.Unmarshal(out, &pkg); err != nil {
		t.Fatal(err)
	}
	const bad = "github.com/mewisme/mew/internal/runner/envexec"
	for _, imp := range pkg.Imports {
		if imp == bad || strings.HasPrefix(imp, bad+"/") {
			t.Errorf("internal/runner must not import %s", imp)
		}
	}
}

func TestGoListEnvexecImportsIsDeterministic(t *testing.T) {
	cmd := exec.Command("go", "list", "-f", "{{join .Imports \"\\n\"}}", "github.com/mewisme/mew/internal/runner/envexec")
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if len(bytes.TrimSpace(out)) == 0 {
		t.Fatal("expected imports")
	}
}
