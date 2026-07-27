package store

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mewisme/m/internal/apperr"
)

const importLockName = ".import.lock"

func importLockPath(pkgDir string) string {
	return filepath.Join(pkgDir, importLockName)
}

func acquireImportLock(pkgDir string) (release func(), err error) {
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		return nil, apperr.Wrap(apperr.Store, "store.import.lock", pkgDir, err)
	}
	lock := importLockPath(pkgDir)
	deadline := time.Now().Add(30 * time.Second)
	for {
		f, err := os.OpenFile(lock, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err == nil {
			_, _ = fmt.Fprintf(f, "%d\n", os.Getpid())
			_ = f.Close()
			return func() { _ = os.Remove(lock) }, nil
		}
		if !os.IsExist(err) {
			return nil, apperr.Wrap(apperr.Store, "store.import.lock", lock, err)
		}
		if time.Now().After(deadline) {
			return nil, apperr.New(apperr.Store, "store.import.lock", lock, "timeout waiting for import lock")
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// HasImportLock reports whether another process holds the per-package import lock.
func HasImportLock(pkgDir string) bool {
	_, err := os.Stat(importLockPath(pkgDir))
	return err == nil
}

// clearImportSlot removes an empty package dir left after releasing the import lock.
func clearImportSlot(pkgDir string) {
	lock := importLockPath(pkgDir)
	_ = os.Remove(lock)
	entries, err := os.ReadDir(pkgDir)
	if err != nil || len(entries) > 0 {
		return
	}
	_ = os.Remove(pkgDir)
}
