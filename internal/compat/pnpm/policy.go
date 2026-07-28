package pnpm

import (
	"github.com/mewisme/mew/internal/lockfile"
)

// Policy owns generation-specific lockfile read/write rules.
type Policy interface {
	Format() string
	ProducerMajor() int
	DefaultLockfileVersion() string
	// StructuralEvidence returns generation-specific markers found in doc.
	StructuralEvidence(doc *Document) []string
	// DetectFromStructure returns detection when structural evidence is definitive.
	DetectFromStructure(doc *Document) (lockfile.Detection, bool)
	// ApplyPackageEncodeFields adds generation-specific package fields during encode.
	ApplyPackageEncodeFields(entry PackageEntry, m map[string]any)
}

// SelectPolicy returns the generation policy for a detection result.
func SelectPolicy(det lockfile.Detection) Policy {
	switch det.Format {
	case FormatV6:
		return v6Policy{}
	case FormatV10:
		return v10Policy{}
	case FormatV11:
		return v11Policy{}
	default:
		return v9Policy{}
	}
}

// DetectFromDocument applies policy-owned structural detection to a parsed document.
func DetectFromDocument(doc *Document) (lockfile.Detection, bool) {
	if doc == nil {
		return lockfile.Detection{}, false
	}
	if IsV6Layout(doc) {
		return v6Policy{}.DetectFromStructure(doc)
	}
	for _, p := range []Policy{v11Policy{}, v10Policy{}, v9Policy{}} {
		if det, ok := p.DetectFromStructure(doc); ok {
			return det, true
		}
	}
	return lockfile.Detection{}, false
}

type v6Policy struct{}

func (v6Policy) Format() string                          { return FormatV6 }
func (v6Policy) ProducerMajor() int                      { return 0 }
func (v6Policy) DefaultLockfileVersion() string          { return "5.4" }
func (v6Policy) StructuralEvidence(_ *Document) []string { return nil }
func (v6Policy) DetectFromStructure(doc *Document) (lockfile.Detection, bool) {
	if !IsV6Layout(doc) {
		return lockfile.Detection{}, false
	}
	return lockfile.Detection{
		Format: FormatV6, Confidence: lockfile.DetectionCertain,
		Evidence: []string{"lockfileVersion=" + doc.LockfileVersion, "layout=v6-flat"},
	}, true
}
func (v6Policy) ApplyPackageEncodeFields(_ PackageEntry, _ map[string]any) {}

type v9Policy struct{}

func (v9Policy) Format() string                 { return FormatV9 }
func (v9Policy) ProducerMajor() int             { return 9 }
func (v9Policy) DefaultLockfileVersion() string { return "9.0" }
func (v9Policy) StructuralEvidence(doc *Document) []string {
	if hasV9ShapeWithoutNewerMarkers(doc) {
		return []string{"layout=importers-snapshots", "no-v10-v11-package-markers"}
	}
	return nil
}
func (v9Policy) DetectFromStructure(_ *Document) (lockfile.Detection, bool) {
	return lockfile.Detection{}, false
}
func (v9Policy) ApplyPackageEncodeFields(_ PackageEntry, _ map[string]any) {}

type v10Policy struct{}

func (v10Policy) Format() string                 { return FormatV10 }
func (v10Policy) ProducerMajor() int             { return 10 }
func (v10Policy) DefaultLockfileVersion() string { return "9.0" }
func (v10Policy) StructuralEvidence(doc *Document) []string {
	if packageHasChecksum(doc) {
		return []string{"package.checksum"}
	}
	return nil
}
func (v10Policy) DetectFromStructure(doc *Document) (lockfile.Detection, bool) {
	if doc.LockfileVersion != "9.0" || !packageHasChecksum(doc) {
		return lockfile.Detection{}, false
	}
	return lockfile.Detection{
		Format: FormatV10, ProducerMajor: 10, Confidence: lockfile.DetectionCertain,
		Evidence: []string{"lockfileVersion=9.0", "marker=package-checksum"},
	}, true
}
func (v10Policy) ApplyPackageEncodeFields(entry PackageEntry, m map[string]any) {
	if entry.Checksum != "" {
		m["checksum"] = entry.Checksum
	}
}

type v11Policy struct{}

func (v11Policy) Format() string                 { return FormatV11 }
func (v11Policy) ProducerMajor() int             { return 11 }
func (v11Policy) DefaultLockfileVersion() string { return "9.0" }
func (v11Policy) StructuralEvidence(doc *Document) []string {
	if packageHasBuildPolicy(doc) {
		return []string{"package.buildPolicy"}
	}
	return nil
}
func (v11Policy) DetectFromStructure(doc *Document) (lockfile.Detection, bool) {
	if doc.LockfileVersion != "9.0" || !packageHasBuildPolicy(doc) {
		return lockfile.Detection{}, false
	}
	return lockfile.Detection{
		Format: FormatV11, ProducerMajor: 11, Confidence: lockfile.DetectionCertain,
		Evidence: []string{"lockfileVersion=9.0", "marker=package-buildPolicy"},
	}, true
}
func (v11Policy) ApplyPackageEncodeFields(entry PackageEntry, m map[string]any) {
	if entry.BuildPolicy != nil {
		m["buildPolicy"] = entry.BuildPolicy
	}
}

func hasV9ShapeWithoutNewerMarkers(doc *Document) bool {
	return len(doc.Importers) > 0 || len(doc.Snapshots) > 0
}

func packageHasChecksum(doc *Document) bool {
	for _, p := range doc.Packages {
		if p.Checksum != "" {
			return true
		}
	}
	return false
}

func packageHasBuildPolicy(doc *Document) bool {
	for _, p := range doc.Packages {
		if p.BuildPolicy != nil {
			return true
		}
	}
	return false
}

// ponytail: policy structs are zero-size; upgrade path is per-generation files under v6/v9/v10/v11.
var _ Policy = v9Policy{}
