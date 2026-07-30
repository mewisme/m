package cli

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRootScriptCompletionGateOff(t *testing.T) {
	t.Setenv("MEW_EXPERIMENTAL_DIRECT_SCRIPTS", "")
	projDir := t.TempDir()
	if err := os.WriteFile(projDir+"/package.json", []byte(`{
  "name": "demo",
  "version": "1.0.0",
  "scripts": { "dev": "echo dev", "build": "echo build" }
}`), 0o644); err != nil {
		t.Fatal(err)
	}

	root := NewMRoot(testBuildInfo())
	g := ownerFlags(root)
	g.cwd = projDir

	names, directive := rootScriptCompletion(root, []string{}, "")
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("directive=%v", directive)
	}
	if len(names) != 0 {
		t.Fatalf("names=%v want empty when gate off", names)
	}
}

func TestRootScriptCompletionGateOnExcludesReserved(t *testing.T) {
	t.Setenv("MEW_EXPERIMENTAL_DIRECT_SCRIPTS", "1")
	projDir := t.TempDir()
	if err := os.WriteFile(projDir+"/package.json", []byte(`{
  "name": "demo",
  "version": "1.0.0",
  "scripts": {
    "dev": "echo dev",
    "build": "echo build",
    "add": "echo add-script"
  }
}`), 0o644); err != nil {
		t.Fatal(err)
	}

	root := NewMRoot(testBuildInfo())
	g := ownerFlags(root)
	g.cwd = projDir

	names, _ := rootScriptCompletion(root, []string{}, "")
	got := strings.Join(names, ",")
	if strings.Contains(got, "add") {
		t.Fatalf("reserved script in root completion: %v", names)
	}
	if !strings.Contains(got, "dev") || !strings.Contains(got, "build") {
		t.Fatalf("names=%v", names)
	}
}

func TestRunCompletionIncludesReservedScript(t *testing.T) {
	projDir := t.TempDir()
	if err := os.WriteFile(projDir+"/package.json", []byte(`{
  "name": "demo",
  "version": "1.0.0",
  "scripts": { "add": "echo add-script", "dev": "echo dev" }
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

	names, _ := scriptNameCompletion(cmd, []string{}, "")
	got := strings.Join(names, ",")
	if !strings.Contains(got, "add") {
		t.Fatalf("reserved script missing from m run completion: %v", names)
	}
}
