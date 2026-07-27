package config_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/config"
	"github.com/mewisme/m/internal/testkit"
)

func TestLoadSpecClonePreservesPaths(t *testing.T) {
	root := t.TempDir()
	custom := filepath.Join(root, "custom.jsonc")
	if err := os.WriteFile(custom, []byte(`{"install.linker":"isolated"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	opts := config.LoadOptions{
		CWD:                  root,
		ProjectRoot:          root,
		ProjectPath:          custom,
		RequireProjectConfig: true,
		Env:                  []string{"MEW_OFFLINE=false"},
		CLI:                  map[string]any{"offline": true},
	}
	spec := config.LoadSpecFromOptions(opts)
	clone := spec.Clone()
	clone.Env[0] = "MEW_OFFLINE=true"
	clone.CLI["offline"] = false
	if spec.Env[0] != "MEW_OFFLINE=false" {
		t.Fatalf("env mutated: %v", spec.Env)
	}
	if spec.CLI["offline"] != true {
		t.Fatalf("cli mutated: %v", spec.CLI)
	}
	reloaded := clone.WithProjectRoot(root)
	if reloaded.ProjectPath != custom {
		t.Fatalf("project path=%q", reloaded.ProjectPath)
	}
	eff, err := config.Load(context.Background(), reloaded.LoadOptions())
	if err != nil {
		t.Fatal(err)
	}
	if config.String(eff, "install.linker", "") != "isolated" {
		t.Fatalf("linker=%q", config.String(eff, "install.linker", ""))
	}
}

func TestLoadSpecClonePreservesInitializedEmptyEnv(t *testing.T) {
	opts := config.LoadOptions{
		CWD:         t.TempDir(),
		ProjectRoot: t.TempDir(),
		Env:         []string{},
		EnvSnapshot: config.NewEnvSnapshot([]string{}, "linux"),
	}
	spec := config.LoadSpecFromOptions(opts)
	if !spec.EnvSnapshot.Initialized() {
		t.Fatal("expected initialized-empty snapshot in spec")
	}
	clone := spec.Clone()
	if !clone.EnvSnapshot.Initialized() {
		t.Fatal("clone should preserve initialized-empty snapshot")
	}
	if _, ok := clone.EnvSnapshot.Lookup("MEW_OFFLINE"); ok {
		t.Fatal("initialized-empty clone should not have MEW_OFFLINE")
	}
}

func TestRequireProjectConfigMissing(t *testing.T) {
	root := t.TempDir()
	missing := filepath.Join(root, "missing.jsonc")
	_, err := config.Load(context.Background(), config.LoadOptions{
		CWD:                  root,
		ProjectRoot:          root,
		ProjectPath:          missing,
		RequireProjectConfig: true,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if apperr.CodeOf(err) != apperr.Config {
		t.Fatalf("code=%s", apperr.CodeOf(err))
	}
}

func TestRequireProjectConfigMalformed(t *testing.T) {
	root := t.TempDir()
	bad := filepath.Join(root, "bad.jsonc")
	if err := os.WriteFile(bad, []byte(`{not json`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := config.Load(context.Background(), config.LoadOptions{
		CWD:                  root,
		ProjectRoot:          root,
		ProjectPath:          bad,
		RequireProjectConfig: true,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if apperr.CodeOf(err) != apperr.Config {
		t.Fatalf("code=%s", apperr.CodeOf(err))
	}
}

func TestDefaultProjectConfigOptional(t *testing.T) {
	testkit.CleanEnv(t)
	root := t.TempDir()
	eff, err := config.Load(context.Background(), config.LoadOptions{
		CWD:         root,
		ProjectRoot: root,
	})
	if err != nil {
		t.Fatal(err)
	}
	if config.String(eff, "install.linker", "") != "auto" {
		t.Fatalf("linker=%q", config.String(eff, "install.linker", ""))
	}
}

func TestRequireGlobalConfigMissing(t *testing.T) {
	root := t.TempDir()
	missing := filepath.Join(root, "global.jsonc")
	_, err := config.Load(context.Background(), config.LoadOptions{
		CWD:                 root,
		ProjectRoot:         root,
		GlobalPath:          missing,
		RequireGlobalConfig: true,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if apperr.CodeOf(err) != apperr.Config {
		t.Fatalf("code=%s", apperr.CodeOf(err))
	}
}
