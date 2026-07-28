package pnpm

import (
	"fmt"
	"strings"

	"github.com/mewisme/mew/internal/compat/pnpm/v9"
	"github.com/mewisme/mew/internal/lockfile"
)

// ValidateSupportedPnpm is the single authority for supported pnpm lock layouts.
// Only lockfileVersion 9.0 with importers/snapshots layout is accepted.
func ValidateSupportedPnpm(doc *Document) error {
	if doc == nil {
		return lockfile.NewUnsupported("pnpm.support", "pnpm-lock.yaml", "empty document")
	}
	ver := strings.TrimSpace(doc.LockfileVersion)
	if ver == "" {
		return lockfile.NewUnsupported("pnpm.support", "pnpm-lock.yaml", "missing lockfileVersion")
	}
	if IsLegacyUnsupported(doc) {
		return rejectLegacy(doc)
	}
	if ver != v9.LockfileVersion {
		return lockfile.NewUnsupported("pnpm.support", "pnpm-lock.yaml",
			fmt.Sprintf("unsupported lockfileVersion %q (only pnpm 9/10/11 with lockfileVersion 9.0 are supported)", ver))
	}
	if len(doc.Importers) == 0 && len(doc.Snapshots) == 0 {
		return lockfile.NewUnsupported("pnpm.support", "pnpm-lock.yaml",
			"malformed pnpm lock: lockfileVersion 9.0 requires importers/snapshots layout")
	}
	return nil
}

// IsLegacyFlatVersion reports pnpm 5–8 flat lockfileVersion strings.
func IsLegacyFlatVersion(ver string) bool {
	ver = strings.TrimSpace(ver)
	if strings.HasPrefix(ver, "5.") {
		return true
	}
	switch ver {
	case "6", "6.0", "7", "7.0", "8", "8.0":
		return true
	}
	if strings.HasPrefix(ver, "6.") || strings.HasPrefix(ver, "7.") || strings.HasPrefix(ver, "8.") {
		return true
	}
	return false
}
