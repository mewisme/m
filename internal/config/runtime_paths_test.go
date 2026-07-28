package config_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/config"
)

func effFromEnv(env []string, goos string) *config.Effective {
	snap := config.NewEnvSnapshot(env, goos)
	return &config.Effective{Values: map[string]config.Value{}, Env: snap}
}

func TestCacheRootFromMewHomeSnapshot(t *testing.T) {
	home := t.TempDir()
	eff := effFromEnv([]string{"MEW_HOME=" + home}, "linux")
	got := config.CacheRoot(eff)
	want := filepath.Join(home, "cache")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestStoreRootFromMewHomeSnapshot(t *testing.T) {
	home := t.TempDir()
	eff := effFromEnv([]string{"MEW_HOME=" + home}, "linux")
	got, err := config.StoreRoot(eff)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, "store")
	absWant, _ := filepath.Abs(want)
	if got != absWant {
		t.Fatalf("got %q want %q", got, absWant)
	}
}

func TestCacheRootMewCacheDirOverridesHome(t *testing.T) {
	home := t.TempDir()
	cacheDir := filepath.Join(home, "custom-cache")
	eff := effFromEnv([]string{
		"MEW_HOME=" + home,
		"MEW_CACHE_DIR=" + cacheDir,
	}, "linux")
	got := config.CacheRoot(eff)
	if got != cacheDir {
		t.Fatalf("got %q", got)
	}
}

func TestCacheRootWindowsLocalAppData(t *testing.T) {
	eff := effFromEnv([]string{
		"LOCALAPPDATA=C:\\Users\\me\\AppData\\Local",
	}, "windows")
	got := config.CacheRoot(eff)
	if !strings.Contains(got, "mew") || !strings.Contains(got, "cache") {
		t.Fatalf("unexpected cache root: %q", got)
	}
}

func TestCacheRootDarwinHome(t *testing.T) {
	eff := effFromEnv([]string{"HOME=/Users/me"}, "darwin")
	got := config.CacheRoot(eff)
	want := filepath.Join("/Users/me", "Library", "Caches", "mew")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestCacheRootLinuxXDG(t *testing.T) {
	eff := effFromEnv([]string{
		"HOME=/home/user",
		"XDG_CACHE_HOME=/xdg/cache",
	}, "linux")
	got := config.CacheRoot(eff)
	want := filepath.Join("/xdg/cache", "mew")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestStoreRootWindowsMixedCaseLocalAppData(t *testing.T) {
	eff := effFromEnv([]string{
		"LocalAppData=C:\\Users\\me\\AppData\\Local",
	}, "windows")
	got, err := config.StoreRoot(eff)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "mew") || !strings.Contains(got, "store") {
		t.Fatalf("unexpected store root: %q", got)
	}
}

func TestUseGlobalStoreFromSnapshot(t *testing.T) {
	eff := effFromEnv([]string{"MEW_EXPERIMENTAL_GLOBAL_STORE=1"}, "linux")
	if !config.UseGlobalStore(eff) {
		t.Fatal("expected global store enabled from snapshot")
	}
}
