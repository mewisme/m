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
	got, err := resolveConfigWriteTarget(configScopeUser, "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Scope != configScopeUser {
		t.Fatalf("scope=%s", got.Scope)
	}
	want := config.GlobalConfigPath()
	if got.Path != want {
		t.Fatalf("path=%q want %q", got.Path, want)
	}
}

func TestResolveConfigWriteTargetUserExplicit(t *testing.T) {
	t.Setenv("MEW_HOME", t.TempDir())
	got, err := resolveConfigWriteTarget(configScopeUser, "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Scope != configScopeUser {
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
	got, err := resolveConfigWriteTarget(configScopeProject, nested)
	if err != nil {
		t.Fatal(err)
	}
	if got.Scope != configScopeProject {
		t.Fatalf("scope=%s", got.Scope)
	}
	want := filepath.Join(root, "m.jsonc")
	if got.Path != want {
		t.Fatalf("path=%q want %q", got.Path, want)
	}
}

func TestResolveConfigWriteTargetLocalNoProject(t *testing.T) {
	dir := t.TempDir()
	_, err := resolveConfigWriteTarget(configScopeProject, dir)
	if err == nil {
		t.Fatal("expected error")
	}
	if apperr.CodeOf(err) != apperr.NotFound {
		t.Fatalf("code=%s err=%v", apperr.CodeOf(err), err)
	}
}

func TestResolveConfigWriteTargetEffectiveRejected(t *testing.T) {
	_, err := resolveConfigWriteTarget(configScopeEffective, t.TempDir())
	if err == nil {
		t.Fatal("expected error for effective scope")
	}
	if apperr.CodeOf(err) != apperr.Usage {
		t.Fatalf("code=%s err=%v", apperr.CodeOf(err), err)
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

func TestConfigWriteFlagsValidateScope(t *testing.T) {
	validScopes := []string{"user", "project", "effective", ""}
	for _, s := range validScopes {
		f := configWriteFlags{scope: s}
		if err := f.validateScope(); err != nil {
			t.Fatalf("scope=%q should be valid: %v", s, err)
		}
	}
	f := configWriteFlags{scope: "invalid"}
	if err := f.validateScope(); err == nil {
		t.Fatal("expected error for invalid scope")
	}
}

func TestConfigWriteFlagsValidateWritable(t *testing.T) {
	validWritable := []string{"user", "project", ""}
	for _, s := range validWritable {
		f := configWriteFlags{scope: s}
		if err := f.validateWritable(); err != nil {
			t.Fatalf("scope=%q should be writable: %v", s, err)
		}
	}
	f := configWriteFlags{scope: "effective"}
	if err := f.validateWritable(); err == nil {
		t.Fatal("expected error for effective scope")
	}
}
