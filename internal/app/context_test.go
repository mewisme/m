package app_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/testkit"
)

func TestNewRespectsCWD(t *testing.T) {
	home := testkit.TempHome(t)
	proj := filepath.Join(home, "proj")
	testkit.CopyFixture(t, "projects/empty-package-json", proj)

	ac, err := app.New(context.Background(), app.Options{CWD: proj})
	if err != nil {
		t.Fatal(err)
	}
	abs, _ := filepath.Abs(proj)
	if ac.CWD != abs {
		t.Fatalf("CWD=%q want %q", ac.CWD, abs)
	}
	if ac.Config == nil {
		t.Fatal("nil config")
	}
}

func TestFromContextRoundTrip(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"x"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	ac, err := app.New(context.Background(), app.Options{CWD: dir, Version: "1.2.3"})
	if err != nil {
		t.Fatal(err)
	}
	ctx := app.WithContext(context.Background(), ac)
	got := app.FromContext(ctx)
	if got == nil || got.Version != "1.2.3" {
		t.Fatalf("%+v", got)
	}
}

func TestNewClassifiesExplicitConfigAgainstProjectRoot(t *testing.T) {
	testkit.CleanEnv(t)
	repo := t.TempDir()
	pkgDir := filepath.Join(repo, "packages", "app")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "package.json"), []byte(`{"name":"mono"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	custom := filepath.Join(repo, "custom.jsonc")
	if err := os.WriteFile(custom, []byte(`{"install.linker":"isolated"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ac, err := app.New(context.Background(), app.Options{CWD: pkgDir, ConfigPath: custom})
	if err != nil {
		t.Fatal(err)
	}
	if ac.ConfigLoadSpec.ProjectPath != custom {
		t.Fatalf("project path=%q want %q", ac.ConfigLoadSpec.ProjectPath, custom)
	}
	if ac.ConfigLoadSpec.ProjectRoot != repo {
		t.Fatalf("project root=%q want %q", ac.ConfigLoadSpec.ProjectRoot, repo)
	}
	if ac.ConfigLoadSpec.RequireProjectConfig != true {
		t.Fatal("expected RequireProjectConfig")
	}
}

func TestNewClassifiesOutsideConfigAsGlobal(t *testing.T) {
	testkit.CleanEnv(t)
	repo := t.TempDir()
	globalCfg := t.TempDir()
	cfg := filepath.Join(globalCfg, "outside.jsonc")
	if err := os.WriteFile(filepath.Join(repo, "package.json"), []byte(`{"name":"proj"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte(`{"offline":true}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ac, err := app.New(context.Background(), app.Options{CWD: repo, ConfigPath: cfg})
	if err != nil {
		t.Fatal(err)
	}
	if ac.ConfigLoadSpec.GlobalPath != cfg {
		t.Fatalf("global path=%q want %q", ac.ConfigLoadSpec.GlobalPath, cfg)
	}
	if ac.ConfigLoadSpec.RequireGlobalConfig != true {
		t.Fatal("expected RequireGlobalConfig")
	}
}

func TestNewFreezesGlobalPathFromEnvSnapshot(t *testing.T) {
	testkit.CleanEnv(t)
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"snap"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfgDir := filepath.Join(root, "cfg")
	env := []string{"MEW_CONFIG_DIR=" + cfgDir}
	ac, err := app.New(context.Background(), app.Options{CWD: root, Env: env})
	if err != nil {
		t.Fatal(err)
	}
	want, _ := filepath.Abs(filepath.Join(cfgDir, "config.jsonc"))
	if ac.ConfigLoadSpec.GlobalPath != want {
		t.Fatalf("global path=%q want %q", ac.ConfigLoadSpec.GlobalPath, want)
	}
}

func TestNewCacheStorePathsFromEnvSnapshot(t *testing.T) {
	testkit.CleanEnv(t)
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"paths"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cacheDir := filepath.Join(root, "my-cache")
	storeDir := filepath.Join(root, "my-store")
	env := []string{
		"MEW_CACHE_DIR=" + cacheDir,
		"MEW_STORE_DIR=" + storeDir,
	}
	ac, err := app.New(context.Background(), app.Options{CWD: root, Env: env})
	if err != nil {
		t.Fatal(err)
	}
	if got := config.String(ac.Config, "cache.dir", ""); got != cacheDir {
		t.Fatalf("cache.dir=%q want %q", got, cacheDir)
	}
	storeGot, err := config.StoreRoot(ac.Config)
	if err != nil {
		t.Fatal(err)
	}
	storeWant, _ := filepath.Abs(storeDir)
	if storeGot != storeWant {
		t.Fatalf("store=%q want %q", storeGot, storeWant)
	}
}

func TestNewPathsStableAfterAmbientEnvChange(t *testing.T) {
	testkit.CleanEnv(t)
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"stable"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cacheDir := filepath.Join(root, "frozen-cache")
	env := []string{"MEW_CACHE_DIR=" + cacheDir}
	ac, err := app.New(context.Background(), app.Options{CWD: root, Env: env})
	if err != nil {
		t.Fatal(err)
	}
	before := config.CacheRoot(ac.Config)
	t.Setenv("MEW_CACHE_DIR", filepath.Join(root, "ambient-changed"))
	after := config.CacheRoot(ac.Config)
	if before != after {
		t.Fatalf("cache root changed after ambient env: %q -> %q", before, after)
	}
	if before != cacheDir {
		t.Fatalf("cache=%q want %q", before, cacheDir)
	}
}

func TestNewEmptyEnvStaysEmpty(t *testing.T) {
	testkit.CleanEnv(t)
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"empty-env"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MEW_CACHE_DIR", filepath.Join(root, "ambient-cache"))
	t.Setenv("MEW_STORE_DIR", filepath.Join(root, "ambient-store"))
	t.Setenv("MEW_OFFLINE", "true")
	t.Setenv("MEW_EXPERIMENTAL_GLOBAL_STORE", "1")
	t.Setenv("NPM_TOKEN", "host-token")

	ac, err := app.New(context.Background(), app.Options{CWD: root, Env: []string{}})
	if err != nil {
		t.Fatal(err)
	}
	if !ac.Config.Env.Initialized() {
		t.Fatal("expected initialized-empty snapshot")
	}
	if v, ok := ac.Config.Env.Lookup("MEW_CACHE_DIR"); ok || v != "" {
		t.Fatalf("MEW_CACHE_DIR=%q ok=%v want empty", v, ok)
	}
	if v, ok := ac.Config.Env.Lookup("MEW_STORE_DIR"); ok || v != "" {
		t.Fatalf("MEW_STORE_DIR=%q ok=%v want empty", v, ok)
	}
	if v, ok := ac.Config.Env.Lookup("MEW_OFFLINE"); ok || v != "" {
		t.Fatalf("MEW_OFFLINE=%q ok=%v want empty", v, ok)
	}
	if v, ok := ac.Config.Env.Lookup("MEW_EXPERIMENTAL_GLOBAL_STORE"); ok || v != "" {
		t.Fatalf("MEW_EXPERIMENTAL_GLOBAL_STORE=%q ok=%v want empty", v, ok)
	}
	if v, ok := ac.Config.Env.Lookup("NPM_TOKEN"); ok || v != "" {
		t.Fatalf("NPM_TOKEN=%q ok=%v want empty", v, ok)
	}
	if config.String(ac.Config, "cache.dir", "") != "" {
		t.Fatalf("cache.dir should use defaults, not ambient")
	}
}

func TestNewNilEnvSnapshotsHostOnce(t *testing.T) {
	testkit.CleanEnv(t)
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"nil-env"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cacheDir := filepath.Join(root, "host-cache")
	t.Setenv("MEW_CACHE_DIR", cacheDir)

	ac, err := app.New(context.Background(), app.Options{CWD: root})
	if err != nil {
		t.Fatal(err)
	}
	if got := config.String(ac.Config, "cache.dir", ""); got != cacheDir {
		t.Fatalf("cache.dir=%q want %q", got, cacheDir)
	}
	t.Setenv("MEW_CACHE_DIR", filepath.Join(root, "changed"))
	if got := config.String(ac.Config, "cache.dir", ""); got != cacheDir {
		t.Fatalf("cache.dir changed after ambient: %q want %q", got, cacheDir)
	}
}

func TestNewConfigSpecStableAfterChdir(t *testing.T) {
	root := t.TempDir()
	other := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"chdir"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	custom := filepath.Join(root, "m.jsonc")
	if err := os.WriteFile(custom, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	ac, err := app.New(context.Background(), app.Options{CWD: root, ConfigPath: "m.jsonc"})
	if err != nil {
		t.Fatal(err)
	}
	storedProject := ac.ConfigLoadSpec.ProjectPath
	storedGlobal := ac.ConfigLoadSpec.GlobalPath
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(other); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
	if ac.ConfigLoadSpec.ProjectPath != storedProject {
		t.Fatalf("ProjectPath changed after chdir: %q -> %q", storedProject, ac.ConfigLoadSpec.ProjectPath)
	}
	if ac.ConfigLoadSpec.GlobalPath != storedGlobal {
		t.Fatalf("GlobalPath changed after chdir: %q -> %q", storedGlobal, ac.ConfigLoadSpec.GlobalPath)
	}
}
