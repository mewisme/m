package pnpm

import (
	"github.com/mewisme/mew/internal/compat/pnpm/v10"
	"github.com/mewisme/mew/internal/compat/pnpm/v11"
	"github.com/mewisme/mew/internal/compat/pnpm/v9"
	"github.com/mewisme/mew/internal/lockfile"
)

// Policy owns generation-specific lockfile read/write rules.
type Policy interface {
	Format() string
	ProducerMajor() int
	DefaultLockfileVersion() string
	StructuralEvidence(doc *Document) []string
	DetectFromStructure(doc *Document) (lockfile.Detection, bool)
	ApplyPackageEncodeFields(entry PackageEntry, m map[string]any)
}

// SelectPolicy returns the generation policy for a detection result (9/10/11 only).
func SelectPolicy(det lockfile.Detection) Policy {
	switch det.Format {
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
	if IsLegacyUnsupported(doc) {
		return LegacyClassifier(doc)
	}
	if !hasV9Shape(doc) {
		return lockfile.Detection{}, false
	}
	for _, p := range []Policy{v11Policy{}, v10Policy{}, v9Policy{}} {
		if det, ok := p.DetectFromStructure(doc); ok {
			return det, true
		}
	}
	return lockfile.Detection{}, false
}

type v9Policy struct{}

func (v9Policy) Format() string                 { return FormatV9 }
func (v9Policy) ProducerMajor() int             { return v9.Major }
func (v9Policy) DefaultLockfileVersion() string { return v9.LockfileVersion }
func (v9Policy) StructuralEvidence(doc *Document) []string {
	if hasV9Shape(doc) {
		return []string{"layout=importers-snapshots", "lockfileVersion=" + v9.LockfileVersion}
	}
	return nil
}
func (v9Policy) DetectFromStructure(_ *Document) (lockfile.Detection, bool) {
	return lockfile.Detection{}, false
}
func (v9Policy) ApplyPackageEncodeFields(_ PackageEntry, _ map[string]any) {}

type v10Policy struct{}

func (v10Policy) Format() string                 { return FormatV10 }
func (v10Policy) ProducerMajor() int             { return v10.Major }
func (v10Policy) DefaultLockfileVersion() string { return v10.LockfileVersion }
func (v10Policy) StructuralEvidence(doc *Document) []string {
	if !hasV9Shape(doc) {
		return nil
	}
	var out []string
	if docHasRootField(doc, v10.PatchedDependenciesField) {
		out = append(out, "root."+v10.PatchedDependenciesField)
	}
	if docHasRootField(doc, v10.ConfigDependenciesField) {
		out = append(out, "root."+v10.ConfigDependenciesField)
	}
	return out
}
func (v10Policy) DetectFromStructure(doc *Document) (lockfile.Detection, bool) {
	if !hasV9Shape(doc) {
		return lockfile.Detection{}, false
	}
	ev := v10Policy{}.StructuralEvidence(doc)
	if len(ev) == 0 {
		return lockfile.Detection{}, false
	}
	return lockfile.Detection{
		Format: FormatV10, ProducerMajor: v10.Major, Confidence: lockfile.DetectionInferred,
		Evidence: append([]string{"lockfileVersion=" + v10.LockfileVersion}, ev...),
	}, true
}
func (v10Policy) ApplyPackageEncodeFields(entry PackageEntry, m map[string]any) {
	if entry.Checksum != "" {
		m[v10.PackageChecksumField] = entry.Checksum
	}
}

type v11Policy struct{}

func (v11Policy) Format() string                 { return FormatV11 }
func (v11Policy) ProducerMajor() int             { return v11.Major }
func (v11Policy) DefaultLockfileVersion() string { return v11.LockfileVersion }
func (v11Policy) StructuralEvidence(doc *Document) []string {
	if !hasV9Shape(doc) {
		return nil
	}
	if onlyBuilt, ok := settingListPresent(doc, v11.OnlyBuiltDependenciesSetting); ok {
		return []string{"settings." + v11.OnlyBuiltDependenciesSetting + "=" + onlyBuilt}
	}
	if ignored, ok := settingListPresent(doc, v11.IgnoredBuiltDependenciesSetting); ok {
		return []string{"settings." + v11.IgnoredBuiltDependenciesSetting + "=" + ignored}
	}
	return nil
}
func (v11Policy) DetectFromStructure(doc *Document) (lockfile.Detection, bool) {
	if !hasV9Shape(doc) {
		return lockfile.Detection{}, false
	}
	ev := v11Policy{}.StructuralEvidence(doc)
	if len(ev) == 0 {
		return lockfile.Detection{}, false
	}
	return lockfile.Detection{
		Format: FormatV11, ProducerMajor: v11.Major, Confidence: lockfile.DetectionInferred,
		Evidence: append([]string{"lockfileVersion=" + v11.LockfileVersion}, ev...),
	}, true
}
func (v11Policy) ApplyPackageEncodeFields(entry PackageEntry, m map[string]any) {
	if entry.BuildPolicy != nil {
		m[v11.BuildPolicyField] = entry.BuildPolicy
	}
}

func hasV9Shape(doc *Document) bool {
	if doc == nil || doc.LockfileVersion != v9.LockfileVersion {
		return false
	}
	return len(doc.Importers) > 0 || len(doc.Snapshots) > 0 || len(doc.Packages) > 0
}

func docHasRootField(doc *Document, field string) bool {
	if doc == nil {
		return false
	}
	if _, ok := doc.Extensions[field]; ok {
		return true
	}
	return false
}

func settingListPresent(doc *Document, key string) (string, bool) {
	if doc == nil || doc.Settings == nil {
		return "", false
	}
	v, ok := doc.Settings[key]
	if !ok {
		return "", false
	}
	switch t := v.(type) {
	case []any:
		if len(t) > 0 {
			return "non-empty", true
		}
	case []string:
		if len(t) > 0 {
			return "non-empty", true
		}
	}
	return "", false
}

var _ Policy = v9Policy{}
