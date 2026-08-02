package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/config"
)

func TestConfigSetDefaultWritesUser(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("MEW_CONFIG_DIR", cfgDir)

	root := NewMRoot(testBuildInfo())
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(errBuf)
	root.SetArgs([]string{"config", "set", "offline", "true"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v stderr=%s", err, errBuf.String())
	}
	path := filepath.Join(cfgDir, "config.jsonc")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"offline": true`) {
		t.Fatalf("user config contents:\n%s", b)
	}
	out := buf.String()
	if !strings.Contains(out, "Scope") || !strings.Contains(out, "user") {
		t.Fatalf("stdout:\n%s", out)
	}
}

func TestConfigSetProjectScope(t *testing.T) {
	proj := t.TempDir()
	if err := os.WriteFile(filepath.Join(proj, "package.json"), []byte(`{"name":"app"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfgDir := t.TempDir()
	t.Setenv("MEW_CONFIG_DIR", cfgDir)

	root := NewMRoot(testBuildInfo())
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"--cwd", proj, "config", "set", "install.linker", "hoisted", "--scope", "project"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(proj, "m.jsonc"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"linker": "hoisted"`) {
		t.Fatalf("project config:\n%s", b)
	}
	if _, err := os.Stat(filepath.Join(cfgDir, "config.jsonc")); !os.IsNotExist(err) {
		t.Fatalf("user config should be untouched: %v", err)
	}
}

func TestConfigSetProjectNoProjectFails(t *testing.T) {
	dir := t.TempDir()
	root := NewMRoot(testBuildInfo())
	root.SetArgs([]string{"--cwd", dir, "config", "set", "offline", "true", "--scope", "project"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if apperr.CodeOf(err) != apperr.NotFound {
		t.Fatalf("code=%s err=%v", apperr.CodeOf(err), err)
	}
}

func TestConfigUnset(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("MEW_CONFIG_DIR", cfgDir)
	path := filepath.Join(cfgDir, "config.jsonc")
	if err := config.SetFile(path, "offline", true); err != nil {
		t.Fatal(err)
	}

	root := NewMRoot(testBuildInfo())
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"config", "unset", "offline"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "offline") {
		t.Fatalf("still present:\n%s", b)
	}
	if !strings.Contains(buf.String(), "Removed offline") {
		t.Fatalf("stdout:\n%s", buf.String())
	}
}

func TestConfigGetJSON(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("MEW_CONFIG_DIR", cfgDir)
	if err := config.SetFile(filepath.Join(cfgDir, "config.jsonc"), "offline", true); err != nil {
		t.Fatal(err)
	}

	root := NewMRoot(testBuildInfo())
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"--output", "json", "config", "get", "offline"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("json: %v out=%s", err, buf.String())
	}
	if doc["source"] != "user" {
		t.Fatalf("source=%v doc=%v", doc["source"], doc)
	}
	if doc["key"] != "offline" {
		t.Fatalf("key=%v", doc["key"])
	}
}

func TestConfigPath(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("MEW_CONFIG_DIR", cfgDir)
	proj := t.TempDir()
	if err := os.WriteFile(filepath.Join(proj, "package.json"), []byte(`{"name":"app"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Default: user scope.
	root := NewMRoot(testBuildInfo())
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"config", "path"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	wantUser := filepath.Join(cfgDir, "config.jsonc")
	if strings.TrimSpace(buf.String()) != wantUser {
		t.Fatalf("path=%q want %q", buf.String(), wantUser)
	}

	// Project scope.
	buf.Reset()
	root = NewMRoot(testBuildInfo())
	root.SetOut(buf)
	root.SetArgs([]string{"--cwd", proj, "config", "path", "--scope", "project"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	wantProj := filepath.Join(proj, "m.jsonc")
	if strings.TrimSpace(buf.String()) != wantProj {
		t.Fatalf("project path=%q want %q", buf.String(), wantProj)
	}

	// --all flag.
	buf.Reset()
	root = NewMRoot(testBuildInfo())
	root.SetOut(buf)
	root.SetArgs([]string{"--cwd", proj, "config", "path", "--all"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, wantUser) || !strings.Contains(out, wantProj) {
		t.Fatalf("path --all:\n%s", out)
	}
}

func TestConfigListShowOrigin(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("MEW_CONFIG_DIR", cfgDir)
	if err := config.SetFile(filepath.Join(cfgDir, "config.jsonc"), "offline", true); err != nil {
		t.Fatal(err)
	}
	root := NewMRoot(testBuildInfo())
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"config", "list", "--show-origin"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "user") {
		t.Fatalf("expected displayed source user:\n%s", out)
	}
}

func TestConfigGetMarkdownThemeDefault(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("MEW_CONFIG_DIR", cfgDir)

	root := NewMRoot(testBuildInfo())
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"config", "get", "ui.theme"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	if got := strings.TrimSpace(buf.String()); got != "auto" {
		t.Fatalf("default ui.theme: got %q, want auto", got)
	}
}

func TestConfigGetMarkdownThemeVerbose(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("MEW_CONFIG_DIR", cfgDir)

	root := NewMRoot(testBuildInfo())
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"config", "get", "ui.theme", "--verbose"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "auto") {
		t.Fatalf("value missing:\n%s", out)
	}
	if !strings.Contains(out, "defaults") && !strings.Contains(out, "Source") {
		t.Fatalf("source info missing:\n%s", out)
	}
}

func TestConfigSetMarkdownThemeUser(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("MEW_CONFIG_DIR", cfgDir)

	root := NewMRoot(testBuildInfo())
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"config", "set", "ui.theme", "dark"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(cfgDir, "config.jsonc")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"dark"`) {
		t.Fatalf("user config:\n%s", b)
	}
	out := buf.String()
	if !strings.Contains(out, "dark") {
		t.Fatalf("stdout:\n%s", out)
	}
}

func TestConfigUnsetMarkdownThemeRestoresDefault(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("MEW_CONFIG_DIR", cfgDir)
	path := filepath.Join(cfgDir, "config.jsonc")
	if err := config.SetFile(path, "ui.theme", "light"); err != nil {
		t.Fatal(err)
	}

	// Verify set
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "light") {
		t.Fatalf("config not set: %s", b)
	}

	root := NewMRoot(testBuildInfo())
	root.SetArgs([]string{"config", "unset", "ui.theme"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	b, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "ui.theme") {
		t.Fatalf("still present:\n%s", b)
	}
}

func TestConfigSetMarkdownThemeProjectRejected(t *testing.T) {
	proj := t.TempDir()
	if err := os.WriteFile(filepath.Join(proj, "package.json"), []byte(`{"name":"app"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfgDir := t.TempDir()
	t.Setenv("MEW_CONFIG_DIR", cfgDir)

	root := NewMRoot(testBuildInfo())
	root.SetArgs([]string{"--cwd", proj, "config", "set", "ui.theme", "light", "--scope", "project"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for project scope on user-scoped key")
	}
	if !strings.Contains(err.Error(), "user-scoped") {
		t.Fatalf("error should mention user-scoped: %v", err)
	}
	if apperr.CodeOf(err) != apperr.Usage {
		t.Fatalf("code=%s, want usage", apperr.CodeOf(err))
	}
}

func TestConfigSetMarkdownThemeInvalidValue(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("MEW_CONFIG_DIR", cfgDir)

	root := NewMRoot(testBuildInfo())
	root.SetArgs([]string{"config", "set", "ui.theme", "pink"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for invalid theme")
	}
	if !strings.Contains(err.Error(), "ui.theme") {
		t.Fatalf("error should mention key: %v", err)
	}
}

func TestConfigSetEffectiveRejected(t *testing.T) {
	root := NewMRoot(testBuildInfo())
	root.SetArgs([]string{"config", "set", "offline", "true", "--scope", "effective"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for effective scope")
	}
}

func TestConfigInvalidScope(t *testing.T) {
	root := NewMRoot(testBuildInfo())
	root.SetArgs([]string{"config", "set", "offline", "true", "--scope", "invalid"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for invalid scope")
	}
}
