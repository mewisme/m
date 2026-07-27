package store

import (
	"path/filepath"
	"strings"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/contentid"
	"github.com/mewisme/m/internal/diagnostics"
)

// PackageKey identifies an immutable unpacked package under packages/<algo>/<hex>/.
type PackageKey struct {
	Algo string
	Hex  string
}

// String returns algo/hex for filesystem layout.
func (k PackageKey) String() string {
	return k.Algo + "/" + k.Hex
}

// Integrity returns the npm SRI form algo-hex.
func (k PackageKey) Integrity() string {
	return k.Algo + "-" + k.Hex
}

// PackageKeyFromIdentity builds a store key from normalized content identity.
func PackageKeyFromIdentity(id contentid.Identity) (PackageKey, error) {
	key := PackageKey{Algo: id.Algo, Hex: id.Hex}
	return key, validateKey(key)
}

// PackageKeyFromIntegrity parses an npm integrity string via contentid.ParseSRI.
func PackageKeyFromIntegrity(integrity string) (PackageKey, error) {
	id, err := contentid.ParseSRI(integrity)
	if err != nil {
		return PackageKey{}, apperr.Wrap(apperr.Store, "store.integrity", integrity, err)
	}
	return PackageKeyFromIdentity(id)
}

// PackageStore holds unpacked packages at <root>/packages/<algo>/<hex>/.
type PackageStore struct {
	Root     string
	Reporter diagnostics.Reporter // optional; index upsert failures are reported here
}

// NewPackageStore returns a package store rooted at dir.
func NewPackageStore(root string) *PackageStore {
	return &PackageStore{Root: root}
}

// PackagePath returns the on-disk directory for key.
func (s *PackageStore) PackagePath(key PackageKey) string {
	if s == nil || s.Root == "" {
		return ""
	}
	return filepath.Join(s.Root, "packages", filepath.FromSlash(key.String()))
}

// StagingDir returns a unique staging path under <store>/.staging/.
func (s *PackageStore) stagingDir(id string) string {
	return filepath.Join(s.Root, ".staging", id)
}

func validateKey(key PackageKey) error {
	if err := contentid.ValidateKey(key.Algo, key.Hex); err != nil {
		return err
	}
	if strings.Contains(key.Algo, "..") || strings.Contains(key.Hex, "..") {
		return apperr.New(apperr.Store, "store.package", key.String(), "invalid key")
	}
	return nil
}
