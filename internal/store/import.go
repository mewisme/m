package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"time"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/archive"
)

// ImportFromTarball extracts tarballPath into the global store keyed by integrity.
// Idempotent: existing verified packages are not re-extracted.
func (s *PackageStore) ImportFromTarball(ctx context.Context, tarballPath, integrity string) (PackageKey, error) {
	if err := ctx.Err(); err != nil {
		return PackageKey{}, err
	}
	if s == nil || s.Root == "" {
		return PackageKey{}, apperr.New(apperr.Store, "store.import", "", "nil store")
	}
	key, err := PackageKeyFromIntegrity(integrity)
	if err != nil {
		return PackageKey{}, err
	}
	if err := validateKey(key); err != nil {
		return PackageKey{}, err
	}

	_, _ = s.CleanupStaleStaging(time.Hour)

	dest := s.PackagePath(key)
	if st, err := os.Stat(dest); err == nil && st.IsDir() {
		if err := s.VerifyPackage(ctx, key); err == nil {
			return key, nil
		}
		_ = quarantinePackage(dest)
	} else if err != nil && !os.IsNotExist(err) {
		return PackageKey{}, apperr.Wrap(apperr.Store, "store.import", dest, err)
	}

	release, err := acquireImportLock(dest)
	if err != nil {
		return PackageKey{}, err
	}
	defer func() {
		release()
		clearImportSlot(dest)
	}()

	if st, err := os.Stat(dest); err == nil && st.IsDir() {
		if err := s.VerifyPackage(ctx, key); err == nil {
			return key, nil
		}
		_ = quarantinePackage(dest)
	}

	if err := os.MkdirAll(filepath.Join(s.Root, "packages"), 0o755); err != nil {
		return PackageKey{}, apperr.Wrap(apperr.Store, "store.import", s.Root, err)
	}

	stageID, err := randomStageID()
	if err != nil {
		return PackageKey{}, err
	}
	stage := s.stagingDir(stageID)
	if err := os.MkdirAll(stage, 0o755); err != nil {
		return PackageKey{}, apperr.Wrap(apperr.Store, "store.import", stage, err)
	}
	defer func() { _ = os.RemoveAll(stage) }()

	if err := archive.Extract(ctx, tarballPath, stage, archive.DefaultOptions()); err != nil {
		return PackageKey{}, err
	}
	pkgJSON := filepath.Join(stage, "package.json")
	if _, err := os.Stat(pkgJSON); err != nil {
		return PackageKey{}, apperr.Wrap(apperr.Store, "store.import", pkgJSON, err)
	}
	if err := writePackageMarker(stage, key); err != nil {
		return PackageKey{}, apperr.Wrap(apperr.Store, "store.import", stage, err)
	}
	_ = makeTreeReadOnly(stage)
	manifest, err := generateTreeManifest(stage)
	if err != nil {
		return PackageKey{}, err
	}
	if err := writeTreeManifest(stage, manifest); err != nil {
		return PackageKey{}, err
	}
	if err := verifyTreeManifest(stage, manifest); err != nil {
		return PackageKey{}, err
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return PackageKey{}, apperr.Wrap(apperr.Store, "store.import", dest, err)
	}
	clearImportSlot(dest)
	if err := os.Rename(stage, dest); err != nil {
		if st, statErr := os.Stat(dest); statErr == nil && st.IsDir() {
			if verifyErr := s.VerifyPackage(ctx, key); verifyErr == nil {
				return key, nil
			}
		}
		return PackageKey{}, apperr.Wrap(apperr.Store, "store.import", dest, err)
	}
	_ = makeTreeReadOnly(dest)

	size, _ := dirSize(dest)
	_ = s.indexUpsert(key, integrity, size)
	return key, nil
}

func randomStageID() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", apperr.Wrap(apperr.Store, "store.import", "stage", err)
	}
	return hex.EncodeToString(b[:]), nil
}

func dirSize(root string) (int64, error) {
	var total int64
	err := filepath.WalkDir(root, func(_ string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if info, err := d.Info(); err == nil {
			total += info.Size()
		}
		return nil
	})
	return total, err
}
