package cli

import (
	"bytes"
	"encoding/json"
	"errors"
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

// TestConfigGetUIThemeDefault pins the raw-scope contract: a key that
// only carries a schema default is not configured in user scope, so the default
// scope reports a typed not-set error rather than silently returning "auto".
func TestConfigGetUIThemeDefault(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("MEW_CONFIG_DIR", cfgDir)

	root := NewMRoot(testBuildInfo())
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"config", "get", "ui.theme"})
	err := root.Execute()
	if err == nil {
		t.Fatalf("unset user key must not return the default, got %q", buf.String())
	}
	if apperr.CodeOf(err) != apperr.Config {
		t.Fatalf("code=%s want %s", apperr.CodeOf(err), apperr.Config)
	}
	if !errors.Is(err, config.ErrNotSet) {
		t.Fatalf("errors.Is(err, config.ErrNotSet) must hold: %v", err)
	}

	// The merged value is still reachable through the effective scope.
	buf.Reset()
	root = NewMRoot(testBuildInfo())
	root.SetOut(buf)
	root.SetArgs([]string{"config", "get", "ui.theme", "--scope", "effective"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(buf.String()); got != "auto" {
		t.Fatalf("effective ui.theme: got %q, want auto", got)
	}
}

func TestConfigGetUIThemeVerbose(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("MEW_CONFIG_DIR", cfgDir)
	if err := config.SetFile(filepath.Join(cfgDir, "config.jsonc"), "ui.theme", "dark"); err != nil {
		t.Fatal(err)
	}

	root := NewMRoot(testBuildInfo())
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"config", "get", "ui.theme", "--verbose"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "dark") {
		t.Fatalf("value missing:\n%s", out)
	}
	for _, want := range []string{"Key", "Scope", "Source", "Type", "Default"} {
		if !strings.Contains(out, want) {
			t.Fatalf("verbose output missing %q:\n%s", want, out)
		}
	}
}

func TestConfigSetUIThemeUser(t *testing.T) {
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

func TestConfigUnsetUIThemeRestoresDefault(t *testing.T) {
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

func TestConfigSetUIThemeProjectRejected(t *testing.T) {
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
	// The message is schema-driven: it names the key, the scopes the key
	// allows, and the scope that was rejected.
	for _, want := range []string{"ui.theme", "user", "project"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error should mention %q: %v", want, err)
		}
	}
	if apperr.CodeOf(err) != apperr.Usage {
		t.Fatalf("code=%s, want usage", apperr.CodeOf(err))
	}
}

func TestConfigSetUIThemeInvalidValue(t *testing.T) {
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

// ── secret redaction ────────────────────────────────────────────

func TestConfigGetSecretRedactedHuman(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("MEW_CONFIG_DIR", cfgDir)
	if err := config.SetFile(filepath.Join(cfgDir, "config.jsonc"), "registry.auth_token_env", "MY_TOKEN"); err != nil {
		t.Fatal(err)
	}

	root := NewMRoot(testBuildInfo())
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"config", "get", "registry.auth_token_env"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "MY_TOKEN") {
		t.Fatalf("secret value leaked in human output:\n%s", out)
	}
	if !strings.Contains(out, "<redacted>") {
		t.Fatalf("secret not redacted in human output:\n%s", out)
	}
}

func TestConfigGetSecretRedactedVerbose(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("MEW_CONFIG_DIR", cfgDir)
	if err := config.SetFile(filepath.Join(cfgDir, "config.jsonc"), "registry.auth_token_env", "MY_TOKEN"); err != nil {
		t.Fatal(err)
	}

	root := NewMRoot(testBuildInfo())
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"config", "get", "registry.auth_token_env", "--verbose"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "MY_TOKEN") {
		t.Fatalf("secret value leaked in verbose output:\n%s", out)
	}
	if !strings.Contains(out, "<redacted>") || !strings.Contains(out, "Is secret") {
		t.Fatalf("secret fields missing in verbose:\n%s", out)
	}
}

func TestConfigGetSecretRedactedJSON(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("MEW_CONFIG_DIR", cfgDir)
	if err := config.SetFile(filepath.Join(cfgDir, "config.jsonc"), "registry.auth_token_env", "MY_TOKEN"); err != nil {
		t.Fatal(err)
	}

	root := NewMRoot(testBuildInfo())
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"--output", "json", "config", "get", "registry.auth_token_env"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("json: %v out=%s", err, buf.String())
	}
	if doc["value"] != "<redacted>" {
		t.Fatalf("secret value not redacted in JSON: %v", doc["value"])
	}
	if doc["is_secret"] != true {
		t.Fatalf("is_secret not true in JSON: %v", doc["is_secret"])
	}
}

func TestConfigListSecretRedactedHuman(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("MEW_CONFIG_DIR", cfgDir)
	if err := config.SetFile(filepath.Join(cfgDir, "config.jsonc"), "registry.auth_token_env", "MY_TOKEN"); err != nil {
		t.Fatal(err)
	}

	root := NewMRoot(testBuildInfo())
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"config", "list", "--prefix", "registry.auth_token_env"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "MY_TOKEN") {
		t.Fatalf("secret value leaked in list human output:\n%s", out)
	}
	if !strings.Contains(out, "<redacted>") {
		t.Fatalf("secret not redacted in list human output:\n%s", out)
	}
}

func TestConfigListSecretRedactedJSON(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("MEW_CONFIG_DIR", cfgDir)
	if err := config.SetFile(filepath.Join(cfgDir, "config.jsonc"), "registry.auth_token_env", "MY_TOKEN"); err != nil {
		t.Fatal(err)
	}

	root := NewMRoot(testBuildInfo())
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"--output", "json", "config", "list", "--prefix", "registry.auth_token_env"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("json: %v out=%s", err, buf.String())
	}
	entries, ok := doc["entries"].([]any)
	if !ok || len(entries) == 0 {
		t.Fatalf("no entries in list JSON: %v", doc)
	}
	entry := entries[0].(map[string]any)
	if entry["value"] != "<redacted>" {
		t.Fatalf("secret value not redacted in list JSON: %v", entry["value"])
	}
	if entry["is_secret"] != true {
		t.Fatalf("is_secret missing in list JSON entry: %v", entry)
	}
}

func TestConfigExplainSecretRedactedHuman(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("MEW_CONFIG_DIR", cfgDir)
	if err := config.SetFile(filepath.Join(cfgDir, "config.jsonc"), "registry.auth_token_env", "MY_TOKEN"); err != nil {
		t.Fatal(err)
	}

	root := NewMRoot(testBuildInfo())
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"config", "explain", "registry.auth_token_env"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "MY_TOKEN") {
		t.Fatalf("secret value leaked in explain human output:\n%s", out)
	}
	if !strings.Contains(out, "<redacted>") {
		t.Fatalf("secret not redacted in explain human output:\n%s", out)
	}
}

func TestConfigExplainSecretRedactedJSON(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("MEW_CONFIG_DIR", cfgDir)
	if err := config.SetFile(filepath.Join(cfgDir, "config.jsonc"), "registry.auth_token_env", "MY_TOKEN"); err != nil {
		t.Fatal(err)
	}

	root := NewMRoot(testBuildInfo())
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"--output", "json", "config", "explain", "registry.auth_token_env"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("json: %v out=%s", err, buf.String())
	}
	if doc["value"] != "<redacted>" {
		t.Fatalf("explain JSON value not redacted: %v", doc["value"])
	}
	if doc["effective_value"] != "<redacted>" {
		t.Fatalf("explain JSON effective_value not redacted: %v", doc["effective_value"])
	}
	if doc["default"] != "<redacted>" {
		t.Fatalf("explain JSON default not redacted: %v", doc["default"])
	}
	layers, ok := doc["layers"].([]any)
	if !ok {
		t.Fatalf("layers missing in explain JSON: %v", doc)
	}
	for _, l := range layers {
		layer := l.(map[string]any)
		if layer["value"] != "<redacted>" {
			t.Fatalf("explain layer value not redacted: %v", layer)
		}
	}
}

func TestConfigSetSecretRedactedHuman(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("MEW_CONFIG_DIR", cfgDir)

	root := NewMRoot(testBuildInfo())
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"config", "set", "registry.auth_token_env", "NEW_TOKEN"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "NEW_TOKEN") || strings.Contains(out, "MY_TOKEN") {
		t.Fatalf("secret value leaked in set human output:\n%s", out)
	}
	if !strings.Contains(out, "<redacted>") {
		t.Fatalf("secret current value not redacted in set:\n%s", out)
	}
}

func TestConfigSetSecretRedactedJSON(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("MEW_CONFIG_DIR", cfgDir)

	root := NewMRoot(testBuildInfo())
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"--output", "json", "config", "set", "registry.auth_token_env", "NEW_TOKEN"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("json: %v out=%s", err, buf.String())
	}
	if doc["current"] != "<redacted>" {
		t.Fatalf("set JSON current not redacted: %v", doc["current"])
	}
	if doc["previous"] != nil {
		t.Fatalf("set JSON previous should be null for unset key, got %v", doc["previous"])
	}
	if doc["changed"] != true {
		t.Fatalf("set JSON changed should be true: %v", doc)
	}
}

func TestConfigUnsetSecretRedactedHuman(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("MEW_CONFIG_DIR", cfgDir)
	if err := config.SetFile(filepath.Join(cfgDir, "config.jsonc"), "registry.auth_token_env", "MY_TOKEN"); err != nil {
		t.Fatal(err)
	}

	root := NewMRoot(testBuildInfo())
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"config", "unset", "registry.auth_token_env"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "MY_TOKEN") {
		t.Fatalf("secret value leaked in unset human output:\n%s", out)
	}
	if !strings.Contains(out, "<redacted>") {
		t.Fatalf("secret not redacted in unset human output:\n%s", out)
	}
}

func TestConfigUnsetSecretRedactedJSON(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("MEW_CONFIG_DIR", cfgDir)
	if err := config.SetFile(filepath.Join(cfgDir, "config.jsonc"), "registry.auth_token_env", "MY_TOKEN"); err != nil {
		t.Fatal(err)
	}

	root := NewMRoot(testBuildInfo())
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"--output", "json", "config", "unset", "registry.auth_token_env"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("json: %v out=%s", err, buf.String())
	}
	if doc["previous"] != "<redacted>" {
		t.Fatalf("unset JSON previous not redacted: %v", doc["previous"])
	}
	if doc["changed"] != true {
		t.Fatalf("unset JSON changed should be true: %v", doc)
	}
}

// ── structured schema checks ────────────────────────────────────

func TestConfigGetJSONSchema(t *testing.T) {
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
	// Must be exactly one valid JSON document.
	out := strings.TrimSpace(buf.String())
	if strings.Count(out, "\n") > 0 {
		t.Fatalf("expected single JSON line, got:\n%s", out)
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("json: %v out=%s", err, out)
	}
	for _, field := range []string{"key", "value", "scope", "source", "configured", "is_default", "is_secret", "type"} {
		if _, ok := doc[field]; !ok {
			t.Fatalf("get JSON missing field %q: %v", field, doc)
		}
	}
	// No ANSI in JSON.
	if strings.Contains(out, "\033") {
		t.Fatalf("ANSI escape in JSON output:\n%s", out)
	}
}

func TestConfigListJSONSchema(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("MEW_CONFIG_DIR", cfgDir)
	if err := config.SetFile(filepath.Join(cfgDir, "config.jsonc"), "offline", true); err != nil {
		t.Fatal(err)
	}

	root := NewMRoot(testBuildInfo())
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"--output", "json", "config", "list", "--prefix", "offline"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	out := strings.TrimSpace(buf.String())
	var doc map[string]any
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("json: %v out=%s", err, out)
	}
	if _, ok := doc["scope"]; !ok {
		t.Fatalf("list JSON missing scope")
	}
	entries, ok := doc["entries"].([]any)
	if !ok {
		t.Fatalf("list JSON missing entries array: %v", doc)
	}
	if len(entries) == 0 {
		t.Fatal("list JSON entries empty")
	}
	entry := entries[0].(map[string]any)
	for _, field := range []string{"key", "value", "scope", "source", "configured", "is_default", "is_secret", "type"} {
		if _, ok := entry[field]; !ok {
			t.Fatalf("list JSON entry missing field %q: %v", field, entry)
		}
	}
	if strings.Contains(out, "\033") {
		t.Fatalf("ANSI escape in JSON output")
	}
}

func TestConfigExplainJSONSchema(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("MEW_CONFIG_DIR", cfgDir)

	root := NewMRoot(testBuildInfo())
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"--output", "json", "config", "explain", "ui.theme"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	out := strings.TrimSpace(buf.String())
	var doc map[string]any
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("json: %v out=%s", err, out)
	}
	for _, field := range []string{"key", "value", "effective_value", "scope", "source", "type", "default", "allowed", "scopes", "description", "is_secret", "layers"} {
		if _, ok := doc[field]; !ok {
			t.Fatalf("explain JSON missing field %q: %v", field, doc)
		}
	}
	layers, ok := doc["layers"].([]any)
	if !ok || len(layers) == 0 {
		t.Fatalf("explain JSON layers missing or empty: %v", doc)
	}
	effectiveCount := 0
	for _, l := range layers {
		layer := l.(map[string]any)
		for _, field := range []string{"source", "value", "configured", "effective"} {
			if _, ok := layer[field]; !ok {
				t.Fatalf("explain layer JSON missing field %q: %v", field, layer)
			}
		}
		if layer["effective"] == true {
			effectiveCount++
		}
	}
	if effectiveCount != 1 {
		t.Fatalf("explain must have exactly one effective layer, got %d", effectiveCount)
	}
	if strings.Contains(out, "\033") {
		t.Fatalf("ANSI escape in JSON output")
	}
}

func TestConfigSetJSONSchema(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("MEW_CONFIG_DIR", cfgDir)

	root := NewMRoot(testBuildInfo())
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"--output", "json", "config", "set", "offline", "true"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	out := strings.TrimSpace(buf.String())
	var doc map[string]any
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("json: %v out=%s", err, out)
	}
	for _, field := range []string{"key", "scope", "path", "previous", "current", "changed"} {
		if _, ok := doc[field]; !ok {
			t.Fatalf("set JSON missing field %q: %v", field, doc)
		}
	}
	if strings.Contains(out, "\033") {
		t.Fatalf("ANSI escape in JSON output")
	}
}

func TestConfigUnsetJSONSchema(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("MEW_CONFIG_DIR", cfgDir)
	if err := config.SetFile(filepath.Join(cfgDir, "config.jsonc"), "offline", true); err != nil {
		t.Fatal(err)
	}

	root := NewMRoot(testBuildInfo())
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"--output", "json", "config", "unset", "offline"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	out := strings.TrimSpace(buf.String())
	var doc map[string]any
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("json: %v out=%s", err, out)
	}
	for _, field := range []string{"key", "scope", "path", "previous", "current", "changed"} {
		if _, ok := doc[field]; !ok {
			t.Fatalf("unset JSON missing field %q: %v", field, doc)
		}
	}
	if strings.Contains(out, "\033") {
		t.Fatalf("ANSI escape in JSON output")
	}
}

// ── semantic parity ─────────────────────────────────────────────

func TestConfigGetHumanJSONSameScope(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("MEW_CONFIG_DIR", cfgDir)
	if err := config.SetFile(filepath.Join(cfgDir, "config.jsonc"), "offline", true); err != nil {
		t.Fatal(err)
	}

	// Human.
	rh := NewMRoot(testBuildInfo())
	bh := new(bytes.Buffer)
	rh.SetOut(bh)
	rh.SetArgs([]string{"config", "get", "offline", "--verbose"})
	if err := rh.Execute(); err != nil {
		t.Fatal(err)
	}

	// JSON.
	rj := NewMRoot(testBuildInfo())
	bj := new(bytes.Buffer)
	rj.SetOut(bj)
	rj.SetArgs([]string{"--output", "json", "config", "get", "offline"})
	if err := rj.Execute(); err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(bj.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	// Both use "user" source — human in verbose output, JSON in source field.
	if !strings.Contains(bh.String(), "user") {
		t.Fatalf("human verbose missing user source:\n%s", bh.String())
	}
	if doc["source"] != "user" {
		t.Fatalf("JSON source mismatch: %v", doc["source"])
	}
}

func TestConfigListHumanJSONSamePrefix(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("MEW_CONFIG_DIR", cfgDir)
	if err := config.SetFile(filepath.Join(cfgDir, "config.jsonc"), "offline", true); err != nil {
		t.Fatal(err)
	}

	// Both use --prefix offline.
	// Human.
	rh := NewMRoot(testBuildInfo())
	bh := new(bytes.Buffer)
	rh.SetOut(bh)
	rh.SetArgs([]string{"config", "list", "--prefix", "offline"})
	if err := rh.Execute(); err != nil {
		t.Fatal(err)
	}

	// JSON.
	rj := NewMRoot(testBuildInfo())
	bj := new(bytes.Buffer)
	rj.SetOut(bj)
	rj.SetArgs([]string{"--output", "json", "config", "list", "--prefix", "offline"})
	if err := rj.Execute(); err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(bj.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	entries := doc["entries"].([]any)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry with prefix filter, got %d", len(entries))
	}
	entry := entries[0].(map[string]any)
	if entry["key"] != "offline" {
		t.Fatalf("wrong key in filtered list: %v", entry["key"])
	}
}

func TestConfigListChangedFilterBothModes(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("MEW_CONFIG_DIR", cfgDir)
	// offline defaults to false; setting to true creates a changed entry.
	if err := config.SetFile(filepath.Join(cfgDir, "config.jsonc"), "offline", true); err != nil {
		t.Fatal(err)
	}

	// Human.
	rh := NewMRoot(testBuildInfo())
	bh := new(bytes.Buffer)
	rh.SetOut(bh)
	rh.SetArgs([]string{"config", "list", "--changed"})
	if err := rh.Execute(); err != nil {
		t.Fatal(err)
	}

	// JSON.
	rj := NewMRoot(testBuildInfo())
	bj := new(bytes.Buffer)
	rj.SetOut(bj)
	rj.SetArgs([]string{"--output", "json", "config", "list", "--changed"})
	if err := rj.Execute(); err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(bj.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	// offline must appear in both.
	if !strings.Contains(bh.String(), "offline") {
		t.Fatalf("changed human output missing offline:\n%s", bh.String())
	}
	entries := doc["entries"].([]any)
	found := false
	for _, e := range entries {
		entry := e.(map[string]any)
		if entry["key"] == "offline" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("changed JSON output missing offline: %v", doc)
	}
}

func TestConfigSetUnsetSameValuesBothModes(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("MEW_CONFIG_DIR", cfgDir)

	// Set in human mode.
	rh := NewMRoot(testBuildInfo())
	bh := new(bytes.Buffer)
	rh.SetOut(bh)
	rh.SetArgs([]string{"config", "set", "ui.theme", "dark"})
	if err := rh.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(bh.String(), "dark") {
		t.Fatalf("set human missing value:\n%s", bh.String())
	}

	// Set in JSON mode.
	rj := NewMRoot(testBuildInfo())
	bj := new(bytes.Buffer)
	rj.SetOut(bj)
	rj.SetArgs([]string{"--output", "json", "config", "set", "ui.theme", "light"})
	if err := rj.Execute(); err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(bj.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if doc["current"] != "light" {
		t.Fatalf("set JSON current mismatch: %v", doc["current"])
	}
	if doc["previous"] != "dark" {
		t.Fatalf("set JSON previous mismatch: %v", doc["previous"])
	}
}

// ── not-set structured output ───────────────────────────────────

func TestConfigGetNotSetJSON(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("MEW_CONFIG_DIR", cfgDir)

	root := NewMRoot(testBuildInfo())
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"--output", "json", "config", "get", "ui.theme"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for unset key")
	}
	// JSON must still be valid even with error exit.
	var doc map[string]any
	if uerr := json.Unmarshal(buf.Bytes(), &doc); uerr != nil {
		t.Fatalf("json: %v out=%s", uerr, buf.String())
	}
	if doc["configured"] != false {
		t.Fatalf("configured should be false for not-set: %v", doc)
	}
	if doc["key"] != "ui.theme" {
		t.Fatalf("key mismatch: %v", doc["key"])
	}
	// effective_value must be present (the schema default).
	if _, ok := doc["effective_value"]; !ok {
		t.Fatalf("effective_value missing in not-set JSON: %v", doc)
	}
}

// ── deterministic ordering ──────────────────────────────────────

func TestConfigListDeterministicOrder(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("MEW_CONFIG_DIR", cfgDir)

	first := func() string {
		root := NewMRoot(testBuildInfo())
		buf := new(bytes.Buffer)
		root.SetOut(buf)
		root.SetArgs([]string{"--output", "json", "config", "list"})
		if err := root.Execute(); err != nil {
			t.Fatal(err)
		}
		return buf.String()
	}
	a := first()
	b := first()
	if a != b {
		t.Fatalf("list JSON order not deterministic:\n%s\nvs\n%s", a, b)
	}
}

func TestConfigExplainDeterministicLayers(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("MEW_CONFIG_DIR", cfgDir)
	if err := config.SetFile(filepath.Join(cfgDir, "config.jsonc"), "ui.theme", "dark"); err != nil {
		t.Fatal(err)
	}

	first := func() string {
		root := NewMRoot(testBuildInfo())
		buf := new(bytes.Buffer)
		root.SetOut(buf)
		root.SetArgs([]string{"--output", "json", "config", "explain", "ui.theme"})
		if err := root.Execute(); err != nil {
			t.Fatal(err)
		}
		return buf.String()
	}
	a := first()
	b := first()
	if a != b {
		t.Fatalf("explain JSON layer order not deterministic:\n%s\nvs\n%s", a, b)
	}
}

// ── presentation mode checks ────────────────────────────────────

func TestConfigGetPlainNoANSI(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("MEW_CONFIG_DIR", cfgDir)
	if err := config.SetFile(filepath.Join(cfgDir, "config.jsonc"), "offline", true); err != nil {
		t.Fatal(err)
	}

	root := NewMRoot(testBuildInfo())
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"--no-color", "config", "get", "offline", "--verbose"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "\033") {
		t.Fatalf("ANSI in plain output:\n%s", buf.String())
	}
}

func TestConfigListPlainNoANSI(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("MEW_CONFIG_DIR", cfgDir)
	if err := config.SetFile(filepath.Join(cfgDir, "config.jsonc"), "offline", true); err != nil {
		t.Fatal(err)
	}

	root := NewMRoot(testBuildInfo())
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"--no-color", "config", "list", "--prefix", "offline"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "\033") {
		t.Fatalf("ANSI in plain list output:\n%s", buf.String())
	}
}
