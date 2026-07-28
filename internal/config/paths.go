package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
)

// ResolveConfigPath resolves a --config path against the invocation working directory.
// Absolute paths are cleaned; relative paths join invocationCWD before absoluting.
func ResolveConfigPath(invocationCWD, configPath string) (string, error) {
	if configPath == "" {
		return "", errors.New("empty config path")
	}
	if filepath.IsAbs(configPath) {
		return filepath.Abs(configPath)
	}
	return filepath.Abs(filepath.Join(invocationCWD, configPath))
}

// IsPathWithin reports whether candidate is lexically inside root (no symlink follow).
func IsPathWithin(root, candidate string) (bool, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return false, err
	}
	rootAbs = filepath.Clean(rootAbs)
	candAbs, err := filepath.Abs(candidate)
	if err != nil {
		return false, err
	}
	candAbs = filepath.Clean(candAbs)

	rel, err := filepath.Rel(rootAbs, candAbs)
	if err != nil {
		return false, nil
	}
	if rel == ".." {
		return false, nil
	}
	sep := string(filepath.Separator)
	if strings.HasPrefix(rel, ".."+sep) {
		return false, nil
	}
	if filepath.IsAbs(rel) {
		return false, nil
	}
	return true, nil
}

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
	snap := envSnap(eff)
	if d, ok := snap.Lookup("MEW_CACHE_DIR"); ok && d != "" {
		return d
	}
	if home, ok := snap.Lookup("MEW_HOME"); ok && home != "" {
		return filepath.Join(home, "cache")
	}
	goos := snap.GOOS()
	if goos == "" {
		goos = runtime.GOOS
	}
	if goos == "windows" {
		base, _ := snap.Lookup("LOCALAPPDATA")
		if base == "" {
			profile, _ := snap.Lookup("USERPROFILE")
			base = filepath.Join(profile, "AppData", "Local")
		}
		return filepath.Join(base, "mew", "cache")
	}
	if goos == "darwin" {
		return filepath.Join(userHomeFromSnap(snap), "Library", "Caches", "mew")
	}
	xdg, ok := snap.Lookup("XDG_CACHE_HOME")
	if !ok || xdg == "" {
		xdg = filepath.Join(userHomeFromSnap(snap), ".cache")
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
	snap := envSnap(eff)
	if d, ok := snap.Lookup("MEW_STORE_DIR"); ok && d != "" {
		return validateStorePath(d, "MEW_STORE_DIR")
	}
	if home, ok := snap.Lookup("MEW_HOME"); ok && home != "" {
		return validateStorePath(filepath.Join(home, "store"), "MEW_HOME")
	}
	goos := snap.GOOS()
	if goos == "" {
		goos = runtime.GOOS
	}
	if goos == "windows" {
		base, _ := snap.Lookup("LOCALAPPDATA")
		if base == "" {
			profile, _ := snap.Lookup("USERPROFILE")
			base = filepath.Join(profile, "AppData", "Local")
		}
		return validateStorePath(filepath.Join(base, "mew", "store"), "default")
	}
	if goos == "darwin" {
		return validateStorePath(filepath.Join(userHomeFromSnap(snap), "Library", "Application Support", "github.com", "mewisme", "mew", "store"), "default")
	}
	xdg, ok := snap.Lookup("XDG_DATA_HOME")
	if !ok || xdg == "" {
		xdg = filepath.Join(userHomeFromSnap(snap), ".local", "share")
	}
	return validateStorePath(filepath.Join(xdg, "github.com", "mewisme", "mew", "store"), "default")
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
	if v, ok := envLookup(eff, "MEW_EXPERIMENTAL_GLOBAL_STORE"); ok && v == "1" {
		return true
	}
	return Bool(eff, "link.use_global_store", false)
}

func envSnap(eff *Effective) EnvSnapshot {
	if eff != nil && eff.Env.Initialized() {
		return eff.Env
	}
	// intentional: ambient env fallback when Effective was built without a snapshot (unit tests).
	return NewEnvSnapshot(os.Environ(), runtime.GOOS)
}

func envLookup(eff *Effective, key string) (string, bool) {
	if eff != nil && eff.Env.Initialized() {
		return eff.Env.Lookup(key)
	}
	// intentional: ambient env fallback when Effective was built without a snapshot (unit tests).
	v := os.Getenv(key)
	return v, v != ""
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
func AuthToken(eff *Effective) string {
	name := String(eff, "registry.auth_token_env", "")
	if name == "" {
		return ""
	}
	if eff == nil || !eff.Env.Initialized() {
		// intentional: zero-value Effective compat for unit tests without a snapshot.
		return os.Getenv(name)
	}
	v, ok := eff.Env.Lookup(name)
	if !ok {
		return ""
	}
	return v
}
