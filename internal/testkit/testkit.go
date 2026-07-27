// Package testkit provides fixtures, clean-home helpers, and local registry stubs.
package testkit

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// ModuleRoot walks up from the working directory to the directory containing go.mod.
func ModuleRoot(t testing.TB) string {
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
			t.Fatal("go.mod not found from working directory")
		}
		dir = parent
	}
}

// FixtureDir resolves fixtures/<rel> under the module root.
func FixtureDir(t testing.TB, rel string) string {
	t.Helper()
	path := filepath.Join(ModuleRoot(t), "fixtures", filepath.FromSlash(rel))
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("fixture %s: %v", rel, err)
	}
	return path
}

// CopyFixture copies fixtures/<rel> into dest (dest must not exist or be empty).
func CopyFixture(t testing.TB, rel, dest string) {
	t.Helper()
	src := FixtureDir(t, rel)
	if err := copyTree(src, dest); err != nil {
		t.Fatalf("CopyFixture(%s -> %s): %v", rel, dest, err)
	}
}

func copyTree(src, dest string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()
	_, err = io.Copy(out, in)
	return err
}
