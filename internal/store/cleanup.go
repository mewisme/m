package store

import (
	"os"
	"path/filepath"
	"time"

	"github.com/mewisme/mew/internal/apperr"
)

// Stable cleanup warning codes surfaced through ImportResult and install results.
const (
	CleanupCodeImportLockRelease = "store_import_lock_release"
	CleanupCodeIndexLockRelease  = "store_index_lock_release"
)

// CleanupStaleStaging removes orphaned directories under <store>/.staging/.
func (s *PackageStore) CleanupStaleStaging(maxAge time.Duration) (int, error) {
	if s == nil || s.Root == "" {
		return 0, apperr.New(apperr.Store, "store.cleanup", "", "nil store")
	}
	stagingRoot := filepath.Join(s.Root, ".staging")
	entries, err := os.ReadDir(stagingRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, apperr.Wrap(apperr.Store, "store.cleanup", stagingRoot, err)
	}
	cutoff := time.Now().Add(-maxAge)
	removed := 0
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		info, err := ent.Info()
		if err != nil {
			continue
		}
		if maxAge > 0 && info.ModTime().After(cutoff) {
			continue
		}
		path := filepath.Join(stagingRoot, ent.Name())
		if err := os.RemoveAll(path); err != nil {
			return removed, apperr.Wrap(apperr.Store, "store.cleanup", path, err)
		}
		removed++
	}
	return removed, nil
}
