package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/mewisme/m/internal/apperr"
)

// String returns a string config value, or def if missing/empty.
func String(eff *Effective, key, def string) string {
	if eff == nil {
		return def
	}
	v, ok := eff.Values[key]
	if !ok {
		return def
	}
	s, ok := v.Raw.(string)
	if !ok || s == "" {
		return def
	}
	return s
}

// Bool returns a bool config value, or def if missing.
func Bool(eff *Effective, key string, def bool) bool {
	if eff == nil {
		return def
	}
	v, ok := eff.Values[key]
	if !ok {
		return def
	}
	b, ok := v.Raw.(bool)
	if !ok {
		return def
	}
	return b
}

// Int returns an int config value, or def if missing.
func Int(eff *Effective, key string, def int) int {
	if eff == nil {
		return def
	}
	v, ok := eff.Values[key]
	if !ok {
		return def
	}
	switch n := v.Raw.(type) {
	case int:
		return n
	case float64:
		return int(n)
	case json.Number:
		i, err := n.Int64()
		if err != nil {
			return def
		}
		return int(i)
	case string:
		i, err := strconv.Atoi(n)
		if err != nil {
			return def
		}
		return i
	default:
		return def
	}
}

// CacheRoot resolves the global cache directory (empty cache.dir → platform default).
func CacheRoot(eff *Effective) string {
	if d := String(eff, "cache.dir", ""); d != "" {
		return d
	}
	if d := os.Getenv("MEW_CACHE_DIR"); d != "" {
		return d
	}
	if home := os.Getenv("MEW_HOME"); home != "" {
		return filepath.Join(home, "cache")
	}
	if runtime.GOOS == "windows" {
		base := os.Getenv("LocalAppData")
		if base == "" {
			base = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local")
		}
		return filepath.Join(base, "mew", "cache")
	}
	if runtime.GOOS == "darwin" {
		return filepath.Join(userHome(), "Library", "Caches", "mew")
	}
	xdg := os.Getenv("XDG_CACHE_HOME")
	if xdg == "" {
		xdg = filepath.Join(userHome(), ".cache")
	}
	return filepath.Join(xdg, "mew")
}

// RegistryMetadataCacheDir is <cache>/registry.
func RegistryMetadataCacheDir(eff *Effective) string {
	return filepath.Join(CacheRoot(eff), "registry")
}

// BlobCacheDir is <cache>/blobs for verified tarball bytes.
func BlobCacheDir(eff *Effective) string {
	return filepath.Join(CacheRoot(eff), "blobs")
}

// StoreRoot resolves the global content store (store.dir → MEW_STORE_DIR → platform default).
// The path is absolute, must not contain .., and is created when missing.
func StoreRoot(eff *Effective) (string, error) {
	if d := String(eff, "store.dir", ""); d != "" {
		return validateStorePath(d, "store.dir")
	}
	if d := os.Getenv("MEW_STORE_DIR"); d != "" {
		return validateStorePath(d, "MEW_STORE_DIR")
	}
	if home := os.Getenv("MEW_HOME"); home != "" {
		return validateStorePath(filepath.Join(home, "store"), "MEW_HOME")
	}
	if runtime.GOOS == "windows" {
		base := os.Getenv("LocalAppData")
		if base == "" {
			base = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local")
		}
		return validateStorePath(filepath.Join(base, "mew", "store"), "default")
	}
	if runtime.GOOS == "darwin" {
		return validateStorePath(filepath.Join(userHome(), "Library", "Application Support", "github.com", "mewisme", "m", "store"), "default")
	}
	xdg := os.Getenv("XDG_DATA_HOME")
	if xdg == "" {
		xdg = filepath.Join(userHome(), ".local", "share")
	}
	return validateStorePath(filepath.Join(xdg, "github.com", "mewisme", "m", "store"), "default")
}

func validateStorePath(p, subject string) (string, error) {
	if p == "" {
		return "", apperr.New(apperr.Config, "config.store", subject, "empty path")
	}
	if strings.Contains(p, "..") {
		return "", apperr.New(apperr.Config, "config.store", subject, "path must not contain ..")
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", apperr.Wrap(apperr.Config, "config.store", subject, err)
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return "", apperr.Wrap(apperr.Config, "config.store", subject, err)
	}
	return abs, nil
}

// UseGlobalStore reports whether the experimental global store + smart linker path is enabled.
func UseGlobalStore(eff *Effective) bool {
	if os.Getenv("MEW_EXPERIMENTAL_GLOBAL_STORE") == "1" {
		return true
	}
	return Bool(eff, "link.use_global_store", false)
}

// ScopeRegistries returns @scope → registry URL from registries.* keys.
func ScopeRegistries(eff *Effective) map[string]string {
	out := map[string]string{}
	if eff == nil {
		return out
	}
	for k, v := range eff.Values {
		if !strings.HasPrefix(k, "registries.") {
			continue
		}
		scope := strings.TrimPrefix(k, "registries.")
		s, ok := v.Raw.(string)
		if !ok || s == "" {
			continue
		}
		out[scope] = s
	}
	return out
}

// AuthToken resolves the bearer token from registry.auth_token_env (env var name).
// Empty when unset. Never logs the token.
func AuthToken(eff *Effective, environ []string) string {
	name := String(eff, "registry.auth_token_env", "")
	if name == "" {
		return ""
	}
	if environ == nil {
		environ = os.Environ()
	}
	for _, e := range environ {
		k, v, ok := strings.Cut(e, "=")
		if ok && k == name {
			return v
		}
	}
	return ""
}
