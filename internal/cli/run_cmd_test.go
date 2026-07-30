package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/jsonfile"
)

func TestRunMissingScript(t *testing.T) {
	modRoot, err := findModuleRoot()
	if err != nil {
		t.Fatal(err)
	}
	fixture := filepath.Join(modRoot, "fixtures", "projects", "empty-package-json")

	root := NewMRoot(testBuildInfo())
	root.SetOut(ioDiscard{})
	root.SetErr(ioDiscard{})
	root.SetArgs([]string{"--cwd", fixture, "run", "no-such-script"})
	err = root.Execute()
	if err == nil {
		t.Fatal("expected missing script error")
	}
	if apperr.CodeOf(err) != apperr.NotFound {
		t.Fatalf("code=%s err=%v", apperr.CodeOf(err), err)
	}
	if apperr.ExitCode(err) != 1 {
		t.Fatalf("exit=%d", apperr.ExitCode(err))
	}
}

func TestRunIfPresentMissingScript(t *testing.T) {
	modRoot, err := findModuleRoot()
	if err != nil {
		t.Fatal(err)
	}
	fixture := filepath.Join(modRoot, "fixtures", "projects", "empty-package-json")

	root := NewMRoot(testBuildInfo())
	root.SetOut(ioDiscard{})
	root.SetErr(ioDiscard{})
	root.SetArgs([]string{"--cwd", fixture, "run", "no-such-script", "--if-present"})
	if code := ExecuteWithContext(root, context.Background()); code != 0 {
		t.Fatalf("exit=%d", code)
	}
}

func TestRunChildExitCodePropagates(t *testing.T) {
	projDir := t.TempDir()
	script := "exit 42"
	if runtime.GOOS == "windows" {
		script = "exit /b 42"
	}
	pkg := map[string]any{
		"name":    "exit-test",
		"version": "1.0.0",
		"scripts": map[string]string{"fail": script},
	}
	raw, err := jsonfile.Marshal(pkg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, "package.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}

	root := NewMRoot(testBuildInfo())
	var errB bytes.Buffer
	root.SetOut(ioDiscard{})
	root.SetErr(&errB)
	root.SetArgs([]string{"--cwd", projDir, "run", "fail"})
	code := ExecuteWithContext(root, context.Background())
	if code != 42 {
		t.Fatalf("exit=%d stderr=%s", code, errB.String())
	}
}

func TestScriptNameCompletion(t *testing.T) {
	projDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projDir, "package.json"), []byte(`{
  "name": "demo",
  "version": "1.0.0",
  "scripts": {
    "build": "echo build",
    "dev": "echo dev"
  }
}`), 0o644); err != nil {
		t.Fatal(err)
	}

	root := NewMRoot(testBuildInfo())
	cmd, _, err := root.Find([]string{"run"})
	if err != nil {
		t.Fatal(err)
	}
	g := ownerFlags(root)
	g.cwd = projDir

	names, directive := scriptNameCompletion(cmd, []string{}, "")
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("directive=%v", directive)
	}
	if len(names) != 2 {
		t.Fatalf("names=%v", names)
	}
	got := strings.Join(names, ",")
	if !strings.Contains(got, "build") || !strings.Contains(got, "dev") {
		t.Fatalf("names=%v", names)
	}
}
