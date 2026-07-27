package store

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/mewisme/m/internal/apperr"
)

// publishReadOnly controls whether ImportFromTarball chmods the published tree.
// Store tests disable this so t.TempDir cleanup can remove imported packages.
var publishReadOnly = true

// SetPublishReadOnly toggles post-import read-only chmod (tests only).
func SetPublishReadOnly(v bool) {
	publishReadOnly = v
}

// makeTreeReadOnly sets best-effort read-only permissions on a published tree.
func makeTreeReadOnly(root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		mode := fs.FileMode(0o444)
		if d.IsDir() {
			mode = 0o555
		}
		if chmodErr := os.Chmod(path, mode); chmodErr != nil {
			// ponytail: best-effort per OS; Windows ACLs may ignore chmod
			_ = chmodErr
		}
		return nil
	})
}

func quarantinePackage(pkgDir string) error {
	if pkgDir == "" {
		return nil
	}
	if _, err := os.Stat(pkgDir); os.IsNotExist(err) {
		return nil
	}
	hex := filepath.Base(pkgDir)
	algo := filepath.Base(filepath.Dir(pkgDir))
	packagesRoot := filepath.Dir(filepath.Dir(pkgDir))
	storeRoot := filepath.Dir(packagesRoot)
	quarantineRoot := filepath.Join(storeRoot, ".quarantine", algo)
	if err := os.MkdirAll(quarantineRoot, 0o755); err != nil {
		return apperr.Wrap(apperr.Store, "store.quarantine", quarantineRoot, err)
	}
	dest := filepath.Join(quarantineRoot, hex)
	if _, err := os.Stat(dest); err == nil {
		_ = os.RemoveAll(dest)
	}
	if err := os.Rename(pkgDir, dest); err != nil {
		return apperr.Wrap(apperr.Store, "store.quarantine", pkgDir, err)
	}
	return nil
}
