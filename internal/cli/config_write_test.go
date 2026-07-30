package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/config"
)

func TestResolveConfigWriteTargetUserDefault(t *testing.T) {
	t.Setenv("MEW_CONFIG_DIR", t.TempDir())
	got, err := resolveConfigWriteTarget(configWriteOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Scope != configWriteUser {
		t.Fatalf("scope=%s", got.Scope)
	}
	want := config.GlobalConfigPath()
	if got.Path != want {
		t.Fatalf("path=%q want %q", got.Path, want)
	}
}

func TestResolveConfigWriteTargetGlobalAlias(t *testing.T) {
	t.Setenv("MEW_HOME", t.TempDir())
	got, err := resolveConfigWriteTarget(configWriteOptions{Global: true})
	if err != nil {
		t.Fatal(err)
	}
	if got.Scope != configWriteUser {
		t.Fatalf("scope=%s", got.Scope)
	}
	if got.Path != config.GlobalConfigPath() {
		t.Fatalf("path=%q", got.Path)
	}
}

func TestResolveConfigWriteTargetLocalProject(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "packages", "a")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"root"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := resolveConfigWriteTarget(configWriteOptions{Local: true, CWD: nested})
	if err != nil {
		t.Fatal(err)
	}
	if got.Scope != configWriteProject {
		t.Fatalf("scope=%s", got.Scope)
	}
	want := filepath.Join(root, "m.jsonc")
	if got.Path != want {
		t.Fatalf("path=%q want %q", got.Path, want)
	}
}

func TestResolveConfigWriteTargetLocalNoProject(t *testing.T) {
	dir := t.TempDir()
	_, err := resolveConfigWriteTarget(configWriteOptions{Local: true, CWD: dir})
	if err == nil {
		t.Fatal("expected error")
	}
	if apperr.CodeOf(err) != apperr.NotFound {
		t.Fatalf("code=%s err=%v", apperr.CodeOf(err), err)
	}
}

func TestResolveConfigWriteTargetFileRelativeToCWD(t *testing.T) {
	cwd := t.TempDir()
	got, err := resolveConfigWriteTarget(configWriteOptions{File: "extra.jsonc", CWD: cwd})
	if err != nil {
		t.Fatal(err)
	}
	if got.Scope != configWriteFile {
		t.Fatalf("scope=%s", got.Scope)
	}
	want, err := filepath.Abs(filepath.Join(cwd, "extra.jsonc"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Path != want {
		t.Fatalf("path=%q want %q", got.Path, want)
	}
}

func TestResolveConfigWriteTargetFileRejectsDirectory(t *testing.T) {
	dir := t.TempDir()
	_, err := resolveConfigWriteTarget(configWriteOptions{File: dir, CWD: t.TempDir()})
	if err == nil {
		t.Fatal("expected directory rejection")
	}
	if apperr.CodeOf(err) != apperr.IO {
		t.Fatalf("code=%s err=%v", apperr.CodeOf(err), err)
	}
}

func TestResolveConfigWriteTargetMutualExclusion(t *testing.T) {
	cases := []configWriteOptions{
		{Local: true, File: "x.jsonc"},
		{Global: true, Local: true},
		{Global: true, File: "x.jsonc"},
	}
	for _, opts := range cases {
		_, err := resolveConfigWriteTarget(opts)
		if err == nil {
			t.Fatalf("expected usage error for %+v", opts)
		}
		if apperr.CodeOf(err) != apperr.Usage {
			t.Fatalf("code=%s for %+v", apperr.CodeOf(err), opts)
		}
	}
}

func TestDisplayConfigSourceMapsGlobalToUser(t *testing.T) {
	if got := displayConfigSource(config.SourceGlobal); got != "user" {
		t.Fatalf("got %q", got)
	}
	if got := displayConfigSource(config.SourceProject); got != "project" {
		t.Fatalf("got %q", got)
	}
	if !strings.EqualFold(displayConfigSource(config.SourceEnv), "env") {
		t.Fatalf("env mapping broken")
	}
}
