package project_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/manifest"
	"github.com/mewisme/mew/internal/project"
)

func TestLockfileHintsPackageManager(t *testing.T) {
	dir := t.TempDir()
	pkg := `{
  "name": "hints-test",
  "version": "1.0.0",
  "packageManager": "pnpm@10.4.0",
  "devEngines": { "packageManager": "pnpm@11.0.0" }
}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkg), 0o644); err != nil {
		t.Fatal(err)
	}
	doc, err := manifest.LoadCached(dir)
	if err != nil {
		t.Fatal(err)
	}
	p := &project.Project{Root: dir, Doc: doc}
	hints := project.LockfileHints(p)
	if hints.PackageManager != "pnpm@10.4.0" {
		t.Fatalf("packageManager=%q", hints.PackageManager)
	}
	if hints.DevEnginesPM != "pnpm@11.0.0" {
		t.Fatalf("devEngines=%q", hints.DevEnginesPM)
	}
}
