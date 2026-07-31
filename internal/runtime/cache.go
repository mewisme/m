package runtime

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/runtime/assets"
)

// extractCache manages versioned, content-addressed runtime asset extraction.
type extractCache struct {
	mu sync.Mutex
}

// cacheExtract ensures the named asset is extracted under cacheDir and returns its path.
// Extraction is atomic via temp file + rename. SHA-256 verified.
func (c *extractCache) cacheEnsure(cacheDir string, entry assets.ManifestEntry) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	dest := filepath.Join(cacheDir, entry.Path)
	if _, err := os.Stat(dest); err == nil {
		return dest, nil
	}

	data, err := assets.ReadAsset(entry.Path)
	if err != nil {
		return "", err
	}

	// verify digest before writing
	got := sha256.Sum256(data)
	if !strings.EqualFold(hex.EncodeToString(got[:]), entry.SHA256) {
		return "", apperr.New(apperr.RuntimeAssetDigest, "runtime.cache", entry.Path,
			fmt.Sprintf("embedded digest mismatch: expected %s", entry.SHA256))
	}

	// atomic write: temp file then rename
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", apperr.Wrap(apperr.RuntimeAssetExtract, "runtime.cache", entry.Path, err)
	}
	tmp := dest + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return "", apperr.Wrap(apperr.RuntimeAssetExtract, "runtime.cache", entry.Path, err)
	}
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return "", apperr.Wrap(apperr.RuntimeAssetExtract, "runtime.cache", entry.Path, err)
	}
	return dest, nil
}

// CacheDir returns the versioned runtime cache directory.
func CacheDir(eff *config.Effective) (string, error) {
	if eff == nil {
		return "", errors.New("nil effective config")
	}
	root := config.CacheRoot(eff)
	m, err := assets.LoadManifest()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "runtime", m.BundleVersion), nil
}

// extractionMu serializes cache population across goroutines.
var extractionMu sync.Mutex

// EnsureAssets extracts all runtime assets to the cache and returns a map of asset name to file path.
func EnsureAssets(eff *config.Effective) (map[string]string, error) {
	if eff == nil {
		return nil, apperr.New(apperr.Internal, "runtime.assets", "", "nil effective config")
	}
	cacheDir, err := CacheDir(eff)
	if err != nil {
		return nil, err
	}

	extractionMu.Lock()
	defer extractionMu.Unlock()

	m, err := assets.LoadManifest()
	if err != nil {
		return nil, err
	}

	paths := make(map[string]string, len(m.Assets))
	cache := &extractCache{}
	for _, entry := range m.AssetsSorted() {
		p, err := cache.cacheEnsure(cacheDir, entry)
		if err != nil {
			return nil, fmt.Errorf("extract %s: %w", entry.Name, err)
		}
		paths[entry.Name] = p
	}
	return paths, nil
}

// VerifyCache re-checks every extracted asset digest against the manifest.
func VerifyCache(eff *config.Effective) error {
	cacheDir, err := CacheDir(eff)
	if err != nil {
		return err
	}
	m, err := assets.LoadManifest()
	if err != nil {
		return err
	}
	for _, entry := range m.AssetsSorted() {
		dest := filepath.Join(cacheDir, entry.Path)
		f, err := os.Open(dest)
		if err != nil {
			return apperr.Wrap(apperr.RuntimeAssetCache, "runtime.verify", entry.Path, err)
		}
		verifyErr := assets.VerifyAsset(f, entry.SHA256)
		f.Close()
		if verifyErr != nil {
			return verifyErr
		}
	}
	return nil
}

// CleanCache removes stale runtime cache directories (not the current bundle).
func CleanCache(eff *config.Effective) error {
	if eff == nil {
		return nil
	}
	root := config.CacheRoot(eff)
	runtimeDir := filepath.Join(root, "runtime")
	entries, err := os.ReadDir(runtimeDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	m, err := assets.LoadManifest()
	if err != nil {
		return err
	}
	currentVersion := m.BundleVersion
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if entry.Name() == currentVersion {
			continue
		}
		_ = os.RemoveAll(filepath.Join(runtimeDir, entry.Name()))
	}
	return nil
}
