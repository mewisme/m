package pnpm

import (
	"strings"

	"github.com/mewisme/mew/internal/lockfile"
)

// IsLegacyUnsupported reports v5–v8 flat layouts that are no longer supported.
func IsLegacyUnsupported(doc *Document) bool {
	if doc == nil {
		return false
	}
	return IsV6Layout(doc)
}

// LegacyClassifier returns detection metadata for unsupported legacy locks (errors only).
func LegacyClassifier(doc *Document) (lockfile.Detection, bool) {
	if !IsLegacyUnsupported(doc) {
		return lockfile.Detection{}, false
	}
	ver := doc.LockfileVersion
	if ver == "" {
		ver = "unknown"
	}
	return lockfile.Detection{
		Format:        FormatV6,
		ProducerMajor: 0,
		Confidence:    lockfile.DetectionCertain,
		Evidence:      []string{"lockfileVersion=" + ver, "layout=v6-flat", "unsupported=legacy"},
	}, true
}

// LegacyUnsupportedError returns a typed rejection for legacy pnpm locks.
func LegacyUnsupportedError(doc *Document) *lockfile.PnpmLegacyUnsupportedError {
	if doc == nil {
		return lockfile.NewPnpmLegacyUnsupported("unknown", "unknown")
	}
	layout := "v6-flat"
	if strings.HasPrefix(doc.LockfileVersion, "5.") {
		layout = "v5-flat"
	} else if doc.LockfileVersion == "6" || doc.LockfileVersion == "6.0" {
		layout = "v6-flat"
	}
	return lockfile.NewPnpmLegacyUnsupported(doc.LockfileVersion, layout)
}

// rejectLegacy returns an error when doc is an unsupported legacy generation.
func rejectLegacy(doc *Document) error {
	if !IsLegacyUnsupported(doc) {
		return nil
	}
	legacy := LegacyUnsupportedError(doc)
	return lockfile.NewUnsupported("pnpm.legacy", "pnpm-lock.yaml", legacy.Error())
}
