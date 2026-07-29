package project_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/project"
)

func TestShrinkwrapPrecedence(t *testing.T) {
	dir := t.TempDir()
	lock := []byte(`{"lockfileVersion":3,"packages":{}}`)
	if err := os.WriteFile(filepath.Join(dir, "package-lock.json"), lock, 0o644); err != nil {
		t.Fatal(err)
	}
	shrink := []byte(`{"lockfileVersion":3,"name":"shrink","packages":{}}`)
	if err := os.WriteFile(filepath.Join(dir, "npm-shrinkwrap.json"), shrink, 0o644); err != nil {
		t.Fatal(err)
	}
	if got := project.IncumbentLockBasename(dir, project.IdentityNPM); got != "npm-shrinkwrap.json" {
		t.Fatalf("basename=%q want npm-shrinkwrap.json", got)
	}
	path, ok := project.IncumbentLockPath(dir, project.IdentityNPM)
	if !ok {
		t.Fatal("expected incumbent lock path")
	}
	data, err := project.ReadLockfileBytes(dir, project.IdentityNPM)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(shrink) {
		t.Fatalf("read wrong lock bytes from %s", path)
	}
}

func TestPackageLockDefaultBasename(t *testing.T) {
	dir := t.TempDir()
	lock := []byte(`{"lockfileVersion":3,"packages":{}}`)
	if err := os.WriteFile(filepath.Join(dir, "package-lock.json"), lock, 0o644); err != nil {
		t.Fatal(err)
	}
	if got := project.IncumbentLockBasename(dir, project.IdentityNPM); got != "package-lock.json" {
		t.Fatalf("basename=%q", got)
	}
}
