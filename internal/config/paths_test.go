package config_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/mewisme/m/internal/config"
)

func TestResolveConfigPathRelative(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(root, "custom.jsonc")
	if err := os.WriteFile(cfg, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := config.ResolveConfigPath(sub, filepath.Join("..", "custom.jsonc"))
	if err != nil {
		t.Fatal(err)
	}
	want, _ := filepath.Abs(cfg)
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestResolveConfigPathAbsolute(t *testing.T) {
	root := t.TempDir()
	cfg := filepath.Join(root, "abs.jsonc")
	if err := os.WriteFile(cfg, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	abs, err := filepath.Abs(cfg)
	if err != nil {
		t.Fatal(err)
	}
	got, err := config.ResolveConfigPath(root, abs)
	if err != nil {
		t.Fatal(err)
	}
	if got != abs {
		t.Fatalf("got %q want %q", got, abs)
	}
}

func TestResolveConfigPathIgnoresProcessCWD(t *testing.T) {
	root := t.TempDir()
	other := t.TempDir()
	cfg := filepath.Join(root, "cfg.jsonc")
	if err := os.WriteFile(cfg, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(other); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	got, err := config.ResolveConfigPath(root, "cfg.jsonc")
	if err != nil {
		t.Fatal(err)
	}
	want, _ := filepath.Abs(cfg)
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestResolveConfigPathDotCleaning(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(root, "m.jsonc")
	if err := os.WriteFile(cfg, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := config.ResolveConfigPath(sub, filepath.Join("..", "..", "m.jsonc"))
	if err != nil {
		t.Fatal(err)
	}
	want, _ := filepath.Abs(cfg)
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestIsPathWithinNested(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "packages", "app")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(root, "custom.jsonc")
	if err := os.WriteFile(cfg, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	within, err := config.IsPathWithin(root, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !within {
		t.Fatal("expected config under monorepo root")
	}
	withinNested, err := config.IsPathWithin(nested, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if withinNested {
		t.Fatal("root-level config should be outside nested package cwd")
	}
}

func TestIsPathWithinSibling(t *testing.T) {
	parent := t.TempDir()
	repoA := filepath.Join(parent, "repo-a")
	repoB := filepath.Join(parent, "repo-b")
	if err := os.MkdirAll(repoA, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(repoB, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(repoB, "global.jsonc")
	if err := os.WriteFile(cfg, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	within, err := config.IsPathWithin(repoA, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if within {
		t.Fatal("sibling repo config should be outside project root")
	}
}

func TestIsPathWithinDotDotFilename(t *testing.T) {
	root := t.TempDir()
	odd := filepath.Join(root, "..notparent")
	if err := os.MkdirAll(odd, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(odd, "cfg.jsonc")
	if err := os.WriteFile(cfg, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	within, err := config.IsPathWithin(root, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !within {
		t.Fatal("segment starting with .. should still be inside root")
	}
}

func TestIsPathWithinWindowsDrive(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-only")
	}
	root := `C:\mew-proj`
	cfg := `C:\mew-proj\m.jsonc`
	within, err := config.IsPathWithin(root, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !within {
		t.Fatal("expected within same drive")
	}
	outside, err := config.IsPathWithin(root, `D:\other\cfg.jsonc`)
	if err != nil {
		t.Fatal(err)
	}
	if outside {
		t.Fatal("expected cross-drive path outside")
	}
}

func TestIsPathWithinUNC(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-only")
	}
	if !strings.HasPrefix(`\\server\share`, `\\`) {
		t.Skip("unc not available")
	}
	within, err := config.IsPathWithin(`\\server\share\proj`, `\\server\share\proj\m.jsonc`)
	if err != nil {
		t.Fatal(err)
	}
	if !within {
		t.Fatal("expected UNC nested path within")
	}
}
