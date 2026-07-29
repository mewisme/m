package bun

import (
	"github.com/mewisme/mew/internal/lockfile"
)

// FieldLossAudit reports bun-specific fields not mapped into the canonical graph.
func FieldLossAudit(doc *Document) lockfile.LossReport {
	report := lockfile.LossReport{SchemaVersion: lockfile.LossReportSchemaVersion, Items: []lockfile.LossItem{}}
	if doc == nil {
		return report
	}
	det := doc.Detection
	if det.Format == "" {
		det = DetectFromDocument(doc)
	}
	for k := range doc.Extensions {
		report.Items = append(report.Items, lockfile.LossItem{
			Field: k, Reason: "top-level extension not mapped to canonical graph",
			SourceFormat: det.Format, Semantic: true, ProducerMajor: det.ProducerMajor, Category: "extension",
		})
	}
	for path, entry := range doc.Workspaces {
		for ek := range entry.Extra {
			report.Items = append(report.Items, lockfile.LossItem{
				Field:        "workspaces." + path + "." + ek,
				Reason:       "workspace field not mapped to canonical graph",
				SourceFormat: det.Format, Semantic: true, ProducerMajor: det.ProducerMajor, Category: "loss",
			})
		}
	}
	return report
}
