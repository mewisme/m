package pnpm

import (
	"sort"
	"strings"

	"github.com/mewisme/mew/internal/compat/pnpm/v10"
	"github.com/mewisme/mew/internal/lockfile"
)

const (
	fieldCategoryMapped    = "mapped"
	fieldCategoryExtension = "extension"
	fieldCategoryLoss      = "loss"

	snapFieldTransitivePeerDeps = "transitivePeerDependencies"
)

var packagePlatformFields = []string{"os", "cpu", "libc"}

var nonRegistryResolutionKeys = []string{
	"tarball", "repo", "directory", "type", "commit", "ref", "url",
}

// FieldLossAudit classifies every known pnpm field and returns migration loss items.
func FieldLossAudit(doc *Document) lockfile.LossReport {
	report := lockfile.LossReport{SchemaVersion: lockfile.LossReportSchemaVersion, Items: []lockfile.LossItem{}}
	if doc == nil {
		return report
	}
	major := doc.Detection.ProducerMajor
	format := doc.Detection.Format
	if format == "" && doc.LockfileVersion == "9.0" {
		format = FormatV9
	}

	for k := range doc.Extensions {
		reason, category := extensionLossReason(k)
		report.Items = append(report.Items, toLossItem(fieldLossSpec{
			path: "pnpm-lock.yaml#" + k, field: k, reason: reason,
			semantic: true, major: major, format: format, category: category,
		}))
	}
	if len(doc.Settings) > 0 {
		for sk := range doc.Settings {
			report.Items = append(report.Items, toLossItem(fieldLossSpec{
				path: "settings." + sk, field: "settings." + sk, reason: "settings not represented in canonical graph",
				semantic: true, major: major, format: format, category: fieldCategoryLoss,
			}))
		}
	}
	for id, sec := range doc.Importers {
		for k := range sec.DependenciesMeta {
			report.Items = append(report.Items, toLossItem(fieldLossSpec{
				path:     "importers." + id + ".dependenciesMeta." + k,
				field:    "importers." + id + ".dependenciesMeta." + k,
				reason:   "importer dependenciesMeta not represented in canonical graph",
				semantic: true, major: major, format: format, category: fieldCategoryLoss,
			}))
		}
		if sec.PublishDirectory != "" {
			report.Items = append(report.Items, toLossItem(fieldLossSpec{
				path:     "importers." + id + ".publishDirectory",
				field:    "importers." + id + ".publishDirectory",
				reason:   "importer publishDirectory not represented in canonical graph",
				semantic: true, major: major, format: format, category: fieldCategoryLoss,
			}))
		}
		for ek := range sec.Extra {
			report.Items = append(report.Items, toLossItem(fieldLossSpec{
				path: "importers." + id + "." + ek, field: "importers." + id + "." + ek,
				reason:   "importer extra field not represented in canonical graph",
				semantic: true, major: major, format: format, category: fieldCategoryLoss,
			}))
		}
	}
	for k, p := range doc.Packages {
		auditPackageFields(k, p, major, format, &report)
	}
	for k, snap := range doc.Snapshots {
		auditSnapshotFields(k, snap, major, format, &report)
	}
	sort.SliceStable(report.Items, func(i, j int) bool { return report.Items[i].Field < report.Items[j].Field })
	return report
}

func extensionLossReason(field string) (reason, category string) {
	switch field {
	case "catalogs":
		return "catalogs not represented in canonical graph", fieldCategoryLoss
	case "overrides":
		return "overrides not represented in canonical graph", fieldCategoryLoss
	case v10.PatchedDependenciesField:
		return "patchedDependencies not represented in canonical graph", fieldCategoryLoss
	case v10.ConfigDependenciesField:
		return "configDependencies not represented in canonical graph", fieldCategoryLoss
	case "time":
		return "lock time metadata not represented in canonical graph", fieldCategoryLoss
	default:
		return "top-level extension not mapped to canonical graph", fieldCategoryExtension
	}
}

func auditPackageFields(pkgKey string, p PackageEntry, major int, format string, report *lockfile.LossReport) {
	if len(p.Engines) > 0 {
		report.Items = append(report.Items, toLossItem(fieldLossSpec{
			path: "packages." + pkgKey + ".engines", field: "packages." + pkgKey + ".engines",
			reason:   "package engines not represented in canonical graph",
			semantic: true, major: major, format: format, category: fieldCategoryLoss,
		}))
	}
	auditPackageResolution(pkgKey, p.Resolution, major, format, report)
	for _, pf := range packagePlatformFields {
		if v, ok := p.Extra[pf]; ok && v != nil {
			report.Items = append(report.Items, toLossItem(fieldLossSpec{
				path: "packages." + pkgKey + "." + pf, field: "packages." + pkgKey + "." + pf,
				reason:   "package " + pf + " constraint not represented in canonical graph",
				semantic: true, major: major, format: format, category: fieldCategoryLoss,
			}))
		}
	}
	if p.Checksum != "" {
		report.Items = append(report.Items, toLossItem(fieldLossSpec{
			path: "packages." + pkgKey + ".checksum", field: "packages." + pkgKey + ".checksum",
			reason:   "package checksum not represented in canonical graph",
			semantic: true, major: major, format: format, category: fieldCategoryLoss,
		}))
	}
	if p.BuildPolicy != nil {
		report.Items = append(report.Items, toLossItem(fieldLossSpec{
			path: "packages." + pkgKey + ".buildPolicy", field: "packages." + pkgKey + ".buildPolicy",
			reason:   "package buildPolicy not represented in canonical graph",
			semantic: true, major: major, format: format, category: fieldCategoryLoss,
		}))
	}
	for fk := range p.Extra {
		if isAuditedPackageExtra(fk) {
			continue
		}
		report.Items = append(report.Items, toLossItem(fieldLossSpec{
			path: "packages." + pkgKey + "." + fk, field: "packages." + pkgKey + "." + fk,
			reason:   "package extra field not represented in canonical graph",
			semantic: true, major: major, format: format, category: fieldCategoryLoss,
		}))
	}
}

func isAuditedPackageExtra(field string) bool {
	for _, pf := range packagePlatformFields {
		if field == pf {
			return true
		}
	}
	return false
}

func auditPackageResolution(pkgKey string, res map[string]any, major int, format string, report *lockfile.LossReport) {
	if len(res) == 0 || isRegistryIntegrityOnly(res) {
		return
	}
	base := "packages." + pkgKey + ".resolution"
	for _, rk := range sortedMapKeys(res) {
		if rk == "integrity" {
			continue
		}
		reason := resolutionLossReason(rk)
		report.Items = append(report.Items, toLossItem(fieldLossSpec{
			path: base + "." + rk, field: base + "." + rk, reason: reason,
			semantic: true, major: major, format: format, category: fieldCategoryLoss,
		}))
	}
}

func resolutionLossReason(key string) string {
	switch key {
	case "tarball":
		return "tarball resolution not represented in canonical graph"
	case "repo":
		return "git resolution not represented in canonical graph"
	case "directory":
		return "directory resolution not represented in canonical graph"
	default:
		if strings.Contains(key, "git") {
			return "git resolution not represented in canonical graph"
		}
		return "non-registry resolution not represented in canonical graph"
	}
}

func isRegistryIntegrityOnly(res map[string]any) bool {
	if len(res) == 0 {
		return true
	}
	if len(res) == 1 {
		_, ok := res["integrity"]
		return ok
	}
	for _, key := range nonRegistryResolutionKeys {
		if _, ok := res[key]; ok {
			return false
		}
	}
	_, hasIntegrity := res["integrity"]
	return hasIntegrity && len(res) == 1
}

func auditSnapshotFields(snapKey string, snap map[string]any, major int, format string, report *lockfile.LossReport) {
	if v, ok := snap[snapFieldTransitivePeerDeps]; ok && v != nil {
		report.Items = append(report.Items, toLossItem(fieldLossSpec{
			path:     "snapshots." + snapKey + "." + snapFieldTransitivePeerDeps,
			field:    "snapshots." + snapKey + "." + snapFieldTransitivePeerDeps,
			reason:   "snapshot transitivePeerDependencies not represented in canonical graph",
			semantic: true, major: major, format: format, category: fieldCategoryLoss,
		}))
	}
	if snapHasOnlyTopology(snap) {
		return
	}
	meta := stripSnapshotTopology(snap)
	delete(meta, snapFieldTransitivePeerDeps)
	if len(meta) > 0 {
		report.Items = append(report.Items, toLossItem(fieldLossSpec{
			path: "snapshots." + snapKey, field: "snapshots." + snapKey,
			reason:   "snapshot metadata not represented in canonical graph",
			semantic: true, major: major, format: format, category: fieldCategoryLoss,
		}))
	}
}

func sortedMapKeys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

type fieldLossSpec struct {
	path, field, reason, format, category string
	semantic                              bool
	major                                 int
}

func toLossItem(item fieldLossSpec) lockfile.LossItem {
	return lockfile.LossItem{
		Field:         item.field,
		Reason:        item.reason,
		SourceFormat:  item.format,
		SourcePath:    item.path,
		Semantic:      item.semantic,
		ProducerMajor: item.major,
		Category:      item.category,
	}
}

// SemanticLossItems returns loss items that block non-dry-run migration.
func SemanticLossItems(report lockfile.LossReport) []lockfile.LossItem {
	out := make([]lockfile.LossItem, 0, len(report.Items))
	for _, item := range report.Items {
		if item.Semantic {
			out = append(out, item)
		}
	}
	return out
}
