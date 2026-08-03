package config_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/config"
)

func effFor(env []string, goos string) *config.Effective {
	return &config.Effective{
		Values: map[string]config.Value{},
		Env:    config.NewEnvSnapshot(env, goos),
	}
}

// MEW_HOME derives config, cache, and store from one root.
func TestMewHomeDerivesAllRoots(t *testing.T) {
	home := t.TempDir()
	snap := config.NewEnvSnapshot([]string{"MEW_HOME=" + home}, "linux")

	if got, want := config.GlobalConfigPathFromEnv(snap), filepath.Join(home, "config", "config.jsonc"); got != want {
		t.Errorf("config: got %q want %q", got, want)
	}
	if got, want := config.CacheRoot(effFor([]string{"MEW_HOME=" + home}, "linux")), filepath.Join(home, "cache"); got != want {
		t.Errorf("cache: got %q want %q", got, want)
	}
	store, err := config.StoreRoot(effFor([]string{"MEW_HOME=" + home}, "linux"))
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, "store"); store != want {
		t.Errorf("store: got %q want %q", store, want)
	}
}

// MEW_CONFIG_DIR takes precedence over MEW_HOME for the config path only.
func TestMewConfigDirPrecedence(t *testing.T) {
	cfgDir := t.TempDir()
	home := t.TempDir()
	snap := config.NewEnvSnapshot([]string{
		"MEW_CONFIG_DIR=" + cfgDir,
		"MEW_HOME=" + home,
	}, "linux")

	got := config.GlobalConfigPathFromEnv(snap)
	if want := filepath.Join(cfgDir, "config.jsonc"); got != want {
		t.Fatalf("MEW_CONFIG_DIR must win: got %q want %q", got, want)
	}
	// Cache still derives from MEW_HOME: config dir is config-only.
	cache := config.CacheRoot(effFor([]string{"MEW_CONFIG_DIR=" + cfgDir, "MEW_HOME=" + home}, "linux"))
	if want := filepath.Join(home, "cache"); cache != want {
		t.Fatalf("cache should follow MEW_HOME: got %q want %q", cache, want)
	}
}

// XDG variables drive the Linux layout.
func TestXDGPaths(t *testing.T) {
	cfg, cache, data := t.TempDir(), t.TempDir(), t.TempDir()
	env := []string{
		"XDG_CONFIG_HOME=" + cfg,
		"XDG_CACHE_HOME=" + cache,
		"XDG_DATA_HOME=" + data,
	}

	if got, want := config.GlobalConfigPathFromEnv(config.NewEnvSnapshot(env, "linux")),
		filepath.Join(cfg, "mew", "config.jsonc"); got != want {
		t.Errorf("config: got %q want %q", got, want)
	}
	if got, want := config.CacheRoot(effFor(env, "linux")), filepath.Join(cache, "mew"); got != want {
		t.Errorf("cache: got %q want %q", got, want)
	}
	store, err := config.StoreRoot(effFor(env, "linux"))
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(data, "github.com", "mewisme", "mew", "store"); store != want {
		t.Errorf("store: got %q want %q", store, want)
	}
}

// Linux falls back to ~/.config, ~/.cache, and ~/.local/share without XDG.
func TestLinuxHomeFallbacks(t *testing.T) {
	home := t.TempDir()
	env := []string{"HOME=" + home}

	if got, want := config.GlobalConfigPathFromEnv(config.NewEnvSnapshot(env, "linux")),
		filepath.Join(home, ".config", "mew", "config.jsonc"); got != want {
		t.Errorf("config: got %q want %q", got, want)
	}
	if got, want := config.CacheRoot(effFor(env, "linux")), filepath.Join(home, ".cache", "mew"); got != want {
		t.Errorf("cache: got %q want %q", got, want)
	}
	store, err := config.StoreRoot(effFor(env, "linux"))
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, ".local", "share", "github.com", "mewisme", "mew", "store"); store != want {
		t.Errorf("store: got %q want %q", store, want)
	}
}

// Windows uses AppData for config and LocalAppData for cache and store.
func TestWindowsAppDataPaths(t *testing.T) {
	roaming, local := t.TempDir(), t.TempDir()
	env := []string{"APPDATA=" + roaming, "LOCALAPPDATA=" + local}

	if got, want := config.GlobalConfigPathFromEnv(config.NewEnvSnapshot(env, "windows")),
		filepath.Join(roaming, "mew", "config.jsonc"); got != want {
		t.Errorf("config: got %q want %q", got, want)
	}
	if got, want := config.CacheRoot(effFor(env, "windows")), filepath.Join(local, "mew", "cache"); got != want {
		t.Errorf("cache: got %q want %q", got, want)
	}
	if runtime.GOOS != "windows" {
		// StoreRoot creates the directory, so only assert the path shape here.
		return
	}
	store, err := config.StoreRoot(effFor(env, "windows"))
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(local, "mew", "store"); store != want {
		t.Errorf("store: got %q want %q", store, want)
	}
}

// Windows falls back to USERPROFILE when AppData vars are absent.
func TestWindowsUserProfileFallback(t *testing.T) {
	profile := t.TempDir()
	env := []string{"USERPROFILE=" + profile}

	if got, want := config.GlobalConfigPathFromEnv(config.NewEnvSnapshot(env, "windows")),
		filepath.Join(profile, "AppData", "Roaming", "mew", "config.jsonc"); got != want {
		t.Errorf("config: got %q want %q", got, want)
	}
	if got, want := config.CacheRoot(effFor(env, "windows")),
		filepath.Join(profile, "AppData", "Local", "mew", "cache"); got != want {
		t.Errorf("cache: got %q want %q", got, want)
	}
}

// macOS uses Library/Caches and Library/Application Support.
func TestDarwinPaths(t *testing.T) {
	home := t.TempDir()
	env := []string{"HOME=" + home}

	if got, want := config.CacheRoot(effFor(env, "darwin")),
		filepath.Join(home, "Library", "Caches", "mew"); got != want {
		t.Errorf("cache: got %q want %q", got, want)
	}
	store, err := config.StoreRoot(effFor(env, "darwin"))
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, "Library", "Application Support", "github.com", "mewisme", "mew", "store")
	if store != want {
		t.Errorf("store: got %q want %q", store, want)
	}
}

// A store left behind by the pre-rename layout must be discovered, not
// orphaned, when the canonical path does not exist yet.
func TestLegacyStoreDiscovered(t *testing.T) {
	data := t.TempDir()
	legacy := filepath.Join(data, "github.com", "mewisme", "m", "store")
	if err := os.MkdirAll(legacy, 0o755); err != nil {
		t.Fatal(err)
	}
	// Put a file in it so it is unmistakably populated user data.
	if err := os.WriteFile(filepath.Join(legacy, "marker"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := config.StoreRoot(effFor([]string{"XDG_DATA_HOME=" + data}, "linux"))
	if err != nil {
		t.Fatal(err)
	}
	if got != legacy {
		t.Fatalf("legacy store not discovered: got %q want %q", got, legacy)
	}
	// Adoption is in place: the legacy data is still there, untouched.
	if _, err := os.Stat(filepath.Join(legacy, "marker")); err != nil {
		t.Fatalf("legacy store data disturbed: %v", err)
	}
}

// When both layouts exist, the canonical path wins.
func TestCanonicalStoreBeatsLegacy(t *testing.T) {
	data := t.TempDir()
	canonical := filepath.Join(data, "github.com", "mewisme", "mew", "store")
	legacy := filepath.Join(data, "github.com", "mewisme", "m", "store")
	for _, d := range []string{canonical, legacy} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	got, err := config.StoreRoot(effFor([]string{"XDG_DATA_HOME=" + data}, "linux"))
	if err != nil {
		t.Fatal(err)
	}
	if got != canonical {
		t.Fatalf("canonical must win: got %q want %q", got, canonical)
	}
}

// An explicit store.dir or MEW_STORE_DIR outranks every default, including
// legacy discovery.
func TestExplicitStoreDirOutranksLegacy(t *testing.T) {
	data := t.TempDir()
	if err := os.MkdirAll(filepath.Join(data, "github.com", "mewisme", "m", "store"), 0o755); err != nil {
		t.Fatal(err)
	}
	explicit := t.TempDir()

	got, err := config.StoreRoot(effFor([]string{
		"XDG_DATA_HOME=" + data,
		"MEW_STORE_DIR=" + explicit,
	}, "linux"))
	if err != nil {
		t.Fatal(err)
	}
	if got != explicit {
		t.Fatalf("MEW_STORE_DIR must win: got %q want %q", got, explicit)
	}
}

// No path helper may hardcode a product directory name outside dirs.go.
func TestNoStrayProductDirLiterals(t *testing.T) {
	for _, f := range []string{"paths.go", "global_path.go"} {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, bad := range []string{`"mew"`, `"config.jsonc"`, `"mewisme"`} {
			if strings.Contains(string(b), bad) {
				t.Errorf("%s hardcodes %s; use the constants in dirs.go", f, bad)
			}
		}
	}
}
