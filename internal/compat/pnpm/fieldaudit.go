package pnpm

import (
	"sort"

	"github.com/mewisme/mew/internal/lockfile"
)

const (
	fieldCategoryMapped    = "mapped"
	fieldCategoryExtension = "extension"
	fieldCategoryLoss      = "loss"
)

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
		report.Items = append(report.Items, toLossItem(fieldLossSpec{
			path: "pnpm-lock.yaml#" + k, field: k, reason: "top-level extension not mapped to canonical graph",
			semantic: true, major: major, format: format, category: fieldCategoryExtension,
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
		if p.Checksum != "" {
			report.Items = append(report.Items, toLossItem(fieldLossSpec{
				path: "packages." + k + ".checksum", field: "packages." + k + ".checksum",
				reason:   "package checksum not represented in canonical graph",
				semantic: true, major: major, format: format, category: fieldCategoryLoss,
			}))
		}
		if p.BuildPolicy != nil {
			report.Items = append(report.Items, toLossItem(fieldLossSpec{
				path: "packages." + k + ".buildPolicy", field: "packages." + k + ".buildPolicy",
				reason:   "package buildPolicy not represented in canonical graph",
				semantic: true, major: major, format: format, category: fieldCategoryLoss,
			}))
		}
		for fk := range p.Extra {
			report.Items = append(report.Items, toLossItem(fieldLossSpec{
				path: "packages." + k + "." + fk, field: "packages." + k + "." + fk,
				reason:   "package extra field not represented in canonical graph",
				semantic: true, major: major, format: format, category: fieldCategoryLoss,
			}))
		}
	}
	for k, snap := range doc.Snapshots {
		if snapHasOnlyTopology(snap) {
			continue
		}
		if len(stripSnapshotTopology(snap)) > 0 {
			report.Items = append(report.Items, toLossItem(fieldLossSpec{
				path: "snapshots." + k, field: "snapshots." + k,
				reason:   "snapshot metadata not represented in canonical graph",
				semantic: true, major: major, format: format, category: fieldCategoryLoss,
			}))
		}
	}
	sort.SliceStable(report.Items, func(i, j int) bool { return report.Items[i].Field < report.Items[j].Field })
	return report
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
