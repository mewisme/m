package config

import (
	"os"
	"path/filepath"
)

// Canonical product directory names. Every path helper derives from these, so a
// directory name appears exactly once in the codebase.
//
// These are user-visible paths: changing one orphans existing data. Any change
// needs a legacy entry below and an ADR (see docs/naming.md).
const (
	// dirProduct is the product directory name used under platform roots
	// (%LocalAppData%\mew, ~/.cache/mew, ~/.config/mew, ~/Library/Caches/mew).
	dirProduct = "mew"

	// dirCache, dirStore, and dirConfig are the sub-roots under MEW_HOME and
	// under the Windows product directory.
	dirCache  = "cache"
	dirStore  = "store"
	dirConfig = "config"

	// fileUserConfig is the user-scope config file name.
	fileUserConfig = "config.jsonc"
)

// vendorStorePath is the vendor-qualified store path used on Linux and macOS,
// e.g. $XDG_DATA_HOME/github.com/mewisme/mew/store.
var vendorStorePath = []string{"github.com", "mewisme", dirProduct, dirStore}

// legacyVendorStorePaths are vendor-qualified store paths this build no longer
// writes but must still discover, so an upgrade does not orphan a populated
// store. Oldest last.
//
// "m" was the module name before the github.com/mewisme/mew migration.
var legacyVendorStorePaths = [][]string{
	{"github.com", "mewisme", "m", dirStore},
}

// productDir joins the product directory under a platform base.
func productDir(base string, sub ...string) string {
	return filepath.Join(append([]string{base, dirProduct}, sub...)...)
}

// vendorDir joins a vendor-qualified path under a platform base.
func vendorDir(base string, parts []string) string {
	return filepath.Join(append([]string{base}, parts...)...)
}

// adoptLegacyStore returns canonical unless it does not exist yet and a legacy
// path under the same base does. Adoption is in place: no data is copied,
// moved, or deleted, so the choice is reversible and cannot lose a store.
//
// ponytail: adopt-in-place rather than migrate-on-startup. Ceiling: a user with
// both paths populated keeps using the canonical one and the legacy store stays
// on disk. Upgrade path: `m store migrate` when the store gets a real
// maintenance surface.
func adoptLegacyStore(base, canonical string) string {
	if pathExists(canonical) {
		return canonical
	}
	for _, legacy := range legacyVendorStorePaths {
		if p := vendorDir(base, legacy); pathExists(p) {
			return p
		}
	}
	return canonical
}

func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
