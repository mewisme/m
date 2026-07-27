package store

import (
	"context"
	"os"
	"path/filepath"

	"github.com/mewisme/m/internal/apperr"
)

const packageMarker = ".mew-package-integrity"

// VerifyPackage checks that key exists and contains a valid package tree.
func (s *PackageStore) VerifyPackage(ctx context.Context, key PackageKey) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.Root == "" {
		return apperr.New(apperr.Store, "store.verify", key.String(), "nil store")
	}
	if err := validateKey(key); err != nil {
		return err
	}
	dir := s.PackagePath(key)
	return verifyPackageDir(dir, key)
}

func verifyPackageDir(dir string, key PackageKey) error {
	st, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return apperr.Wrap(apperr.Store, "store.verify", key.String(), err)
		}
		return apperr.Wrap(apperr.Store, "store.verify", dir, err)
	}
	if !st.IsDir() {
		return apperr.New(apperr.Store, "store.verify", dir, "not a directory")
	}
	pkgJSON := filepath.Join(dir, "package.json")
	if _, err := os.Stat(pkgJSON); err != nil {
		return apperr.Wrap(apperr.Store, "store.verify", pkgJSON, err)
	}
	marker := filepath.Join(dir, packageMarker)
	if data, err := os.ReadFile(marker); err == nil {
		want := key.Integrity()
		if string(data) != want {
			return apperr.New(apperr.Store, "store.verify", key.String(), "integrity marker mismatch")
		}
		return nil
	}
	// ponytail: legacy imports without marker pass on package.json presence only
	return nil
}

// writePackageMarker records integrity after successful publish.
func writePackageMarker(dir string, key PackageKey) error {
	return os.WriteFile(filepath.Join(dir, packageMarker), []byte(key.Integrity()), 0o644)
}
