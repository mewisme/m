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

func TestConfigSetLocalWritesProject(t *testing.T) {
	proj := t.TempDir()
	if err := os.WriteFile(filepath.Join(proj, "package.json"), []byte(`{"name":"app"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfgDir := t.TempDir()
	t.Setenv("MEW_CONFIG_DIR", cfgDir)

	root := NewMRoot(testBuildInfo())
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"--cwd", proj, "config", "set", "install.linker", "hoisted", "--local"})
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

func TestConfigSetLocalNoProjectFails(t *testing.T) {
	dir := t.TempDir()
	root := NewMRoot(testBuildInfo())
	root.SetArgs([]string{"--cwd", dir, "config", "set", "offline", "true", "--local"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if apperr.CodeOf(err) != apperr.NotFound {
		t.Fatalf("code=%s err=%v", apperr.CodeOf(err), err)
	}
}

func TestConfigSetFileExplicit(t *testing.T) {
	cwd := t.TempDir()
	root := NewMRoot(testBuildInfo())
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"--cwd", cwd, "config", "set", "offline", "true", "--file", "custom.jsonc"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(cwd, "custom.jsonc"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"offline": true`) {
		t.Fatalf("file contents:\n%s", b)
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

func TestConfigGetSourceJSON(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("MEW_CONFIG_DIR", cfgDir)
	if err := config.SetFile(filepath.Join(cfgDir, "config.jsonc"), "offline", true); err != nil {
		t.Fatal(err)
	}

	root := NewMRoot(testBuildInfo())
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"--output", "json", "config", "get", "offline", "--source"})
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

func TestConfigPathAndPaths(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("MEW_CONFIG_DIR", cfgDir)
	proj := t.TempDir()
	if err := os.WriteFile(filepath.Join(proj, "package.json"), []byte(`{"name":"app"}`), 0o644); err != nil {
		t.Fatal(err)
	}

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

	buf.Reset()
	root = NewMRoot(testBuildInfo())
	root.SetOut(buf)
	root.SetArgs([]string{"--cwd", proj, "config", "path", "--local"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	wantProj := filepath.Join(proj, "m.jsonc")
	if strings.TrimSpace(buf.String()) != wantProj {
		t.Fatalf("local path=%q want %q", buf.String(), wantProj)
	}

	buf.Reset()
	root = NewMRoot(testBuildInfo())
	root.SetOut(buf)
	root.SetArgs([]string{"--cwd", proj, "config", "paths"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, wantUser) || !strings.Contains(out, wantProj) {
		t.Fatalf("paths:\n%s", out)
	}
}

func TestConfigFlagConflicts(t *testing.T) {
	root := NewMRoot(testBuildInfo())
	root.SetArgs([]string{"config", "set", "offline", "true", "--local", "--global"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected conflict")
	}
	if apperr.CodeOf(err) != apperr.Usage {
		t.Fatalf("code=%s err=%v", apperr.CodeOf(err), err)
	}
}

func TestConfigListSourcesDisplaysUser(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("MEW_CONFIG_DIR", cfgDir)
	if err := config.SetFile(filepath.Join(cfgDir, "config.jsonc"), "offline", true); err != nil {
		t.Fatal(err)
	}
	root := NewMRoot(testBuildInfo())
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"config", "list", "--sources"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "user") {
		t.Fatalf("expected displayed source user:\n%s", out)
	}
	if strings.Contains(out, "\tglobal\t") || strings.Contains(out, " global ") {
		// table may still contain the word in paths; require SOURCE column mapped
		lines := strings.Split(out, "\n")
		for _, line := range lines {
			if strings.Contains(line, "offline") && strings.Contains(line, "global") && !strings.Contains(line, "user") {
				t.Fatalf("offline row still shows global:\n%s", line)
			}
		}
	}
}
