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
	"github.com/mewisme/m/internal/contentid"
)

// ImportResult is the outcome of ImportFromTarball.
type ImportResult struct {
	Key             PackageKey
	CleanupWarnings []string
}

// ImportFromTarball extracts tarballPath into the global store keyed by integrity.
// Idempotent: existing verified packages are not re-extracted.
func (s *PackageStore) ImportFromTarball(ctx context.Context, tarballPath string, id contentid.Identity) (ImportResult, error) {
	if err := ctx.Err(); err != nil {
		return ImportResult{}, err
	}
	if s == nil || s.Root == "" {
		return ImportResult{}, apperr.New(apperr.Store, "store.import", "", "nil store")
	}
	key, err := PackageKeyFromIdentity(id)
	if err != nil {
		return ImportResult{}, err
	}

	_, _ = s.CleanupStaleStaging(time.Hour)

	dest := s.PackagePath(key)
	if st, err := os.Stat(dest); err == nil && st.IsDir() {
		if err := s.VerifyPackage(ctx, key); err == nil {
			return ImportResult{Key: key}, nil
		}
	}

	release, err := acquireImportLock(ctx, s.Root, key)
	if err != nil {
		return ImportResult{}, err
	}
	released := false
	defer func() {
		if !released && release != nil {
			_ = release()
		}
	}()

	if st, err := os.Stat(dest); err == nil && st.IsDir() {
		if err := s.VerifyPackage(ctx, key); err == nil {
			return s.finishImportLocked(key, release, &released)
		}
		_ = quarantinePackage(dest)
	} else if err != nil && !os.IsNotExist(err) {
		return ImportResult{}, apperr.Wrap(apperr.Store, "store.import", dest, err)
	}

	if err := os.MkdirAll(filepath.Join(s.Root, "packages"), 0o755); err != nil {
		return ImportResult{}, apperr.Wrap(apperr.Store, "store.import", s.Root, err)
	}

	stageID, err := randomStageID()
	if err != nil {
		return ImportResult{}, err
	}
	stage := s.stagingDir(stageID)
	if err := os.MkdirAll(stage, 0o755); err != nil {
		return ImportResult{}, apperr.Wrap(apperr.Store, "store.import", stage, err)
	}
	defer func() { _ = os.RemoveAll(stage) }()

	if err := archive.Extract(ctx, tarballPath, stage, archive.DefaultOptions()); err != nil {
		return ImportResult{}, err
	}
	pkgJSON := filepath.Join(stage, "package.json")
	if _, err := os.Stat(pkgJSON); err != nil {
		return ImportResult{}, apperr.Wrap(apperr.Store, "store.import", pkgJSON, err)
	}
	if err := writePackageMarker(stage, key); err != nil {
		return ImportResult{}, apperr.Wrap(apperr.Store, "store.import", stage, err)
	}
	if err := makeTreeReadOnly(stage); err != nil {
		return ImportResult{}, apperr.Wrap(apperr.Store, "store.import", stage, err)
	}
	manifest, err := generateTreeManifest(stage)
	if err != nil {
		return ImportResult{}, err
	}
	if err := os.Chmod(stage, 0o755); err != nil {
		return ImportResult{}, apperr.Wrap(apperr.Store, "store.import", stage, err)
	}
	if err := writeTreeManifest(stage, manifest); err != nil {
		return ImportResult{}, err
	}
	if err := makeTreeReadOnly(stage); err != nil {
		return ImportResult{}, apperr.Wrap(apperr.Store, "store.import", stage, err)
	}
	if err := verifyTreeManifest(stage, manifest); err != nil {
		return ImportResult{}, err
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return ImportResult{}, apperr.Wrap(apperr.Store, "store.import", dest, err)
	}
	if err := os.Rename(stage, dest); err != nil {
		if st, statErr := os.Stat(dest); statErr == nil && st.IsDir() {
			if verifyErr := s.VerifyPackage(ctx, key); verifyErr == nil {
				return s.finishImportLocked(key, release, &released)
			}
		}
		return ImportResult{}, apperr.Wrap(apperr.Store, "store.import", dest, err)
	}

	if err := s.VerifyPackage(ctx, key); err != nil {
		return ImportResult{}, err
	}

	size, _ := dirSize(dest)
	s.indexUpsertOrWarn(key, key.Integrity(), size)
	return s.finishImportLocked(key, release, &released)
}

func (s *PackageStore) finishImportLocked(key PackageKey, release func() error, released *bool) (ImportResult, error) {
	result := ImportResult{Key: key}
	if release == nil {
		return result, nil
	}
	*released = true
	if err := release(); err != nil {
		result.CleanupWarnings = append(result.CleanupWarnings, err.Error())
		s.warnImportLockRelease(err)
	}
	return result, nil
}

func (s *PackageStore) warnImportLockRelease(err error) {
	if s == nil || err == nil {
		return
	}
	s.warnMaintenance("store import lock release failed", err)
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
