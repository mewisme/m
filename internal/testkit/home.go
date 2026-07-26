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

	// Go module cache files are read-only; keep them outside t.TempDir() so
	// cleanup does not fail after go build/run in isolated-home tests.
	goRoot, err := os.MkdirTemp("", "mew-go-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { removeReadonlyTree(goRoot) })
	t.Setenv("GOMODCACHE", filepath.Join(goRoot, "mod"))
	t.Setenv("GOCACHE", filepath.Join(goRoot, "cache"))

	return CleanEnvInfo{Home: home, CacheDir: cache, StoreDir: store, ConfigDir: config}
}

func removeReadonlyTree(root string) {
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err == nil && d != nil {
			_ = os.Chmod(path, 0o700)
		}
		return nil
	})
	_ = os.RemoveAll(root)
}
