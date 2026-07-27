package store

import (
	"path/filepath"
	"strings"

	"github.com/mewisme/m/internal/apperr"
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

// PackageKeyFromIntegrity parses an npm integrity string (algo-hex).
func PackageKeyFromIntegrity(integrity string) (PackageKey, error) {
	integrity = strings.TrimSpace(integrity)
	algo, hex, ok := strings.Cut(integrity, "-")
	if !ok || algo == "" || hex == "" {
		return PackageKey{}, apperr.New(apperr.Store, "store.integrity", integrity, "invalid integrity")
	}
	return PackageKey{Algo: strings.ToLower(algo), Hex: strings.ToLower(hex)}, validateKey(PackageKey{Algo: strings.ToLower(algo), Hex: strings.ToLower(hex)})
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
	if key.Algo == "" || key.Hex == "" {
		return apperr.New(apperr.Store, "store.package", key.String(), "invalid key")
	}
	if strings.Contains(key.Algo, "..") || strings.Contains(key.Hex, "..") {
		return apperr.New(apperr.Store, "store.package", key.String(), "invalid key")
	}
	return nil
}
