package testkit

import (
	"os"
	"path/filepath"
	"testing"
)

// CleanEnvInfo holds isolated paths for a clean-home test.
type CleanEnvInfo struct {
	Home      string
	CacheDir  string
	StoreDir  string
	ConfigDir string
}

// TempHome creates an isolated home directory and sets HOME, USERPROFILE, XDG_*, and MEW_* vars.
func TempHome(t testing.TB) string {
	t.Helper()
	return CleanEnv(t).Home
}

// CleanEnv creates a temp home and redirects all Mew and XDG state under it.
func CleanEnv(t testing.TB) CleanEnvInfo {
	t.Helper()
	home := t.TempDir()
	cache := filepath.Join(home, ".cache", "mew")
	store := filepath.Join(home, ".local", "share", "mew", "store")
	config := filepath.Join(home, ".config", "mew")
	for _, d := range []string{cache, store, config} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, ".local", "state"))
	t.Setenv("MEW_HOME", home)
	t.Setenv("MEW_CACHE_DIR", cache)
	t.Setenv("MEW_STORE_DIR", store)
	t.Setenv("MEW_CONFIG_DIR", config)
	return CleanEnvInfo{Home: home, CacheDir: cache, StoreDir: store, ConfigDir: config}
}
