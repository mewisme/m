package npm

import (
	"github.com/mewisme/mew/internal/lockfile"
)

var auditPackageExtraFields = []string{
	"funding", "cpu", "os", "deprecated", "license", "engines", "bin", "hasInstallScript",
}

// FieldLossAudit reports npm-specific fields not mapped into the canonical graph.
func FieldLossAudit(doc *Document) lockfile.LossReport {
	report := lockfile.LossReport{SchemaVersion: lockfile.LossReportSchemaVersion, Items: []lockfile.LossItem{}}
	if doc == nil {
		return report
	}
	format := doc.Detection.Format
	major := doc.Detection.ProducerMajor
	if format == "" {
		format = DetectFromDocument(doc).Format
		major = doc.LockfileVersion
	}

	for k := range doc.Extensions {
		report.Items = append(report.Items, lockfile.LossItem{
			Field: k, Reason: "top-level extension not mapped to canonical graph",
			SourceFormat: format, Semantic: true, ProducerMajor: major, Category: "extension",
		})
	}
	if len(doc.Dependencies) > 0 {
		report.Items = append(report.Items, lockfile.LossItem{
			Field: "dependencies", Reason: "legacy nested dependencies tree preserved separately from packages map",
			SourceFormat: format, Semantic: false, ProducerMajor: major, Category: "mapped",
		})
	}
	for path, entry := range doc.Packages {
		for _, field := range auditPackageExtraFields {
			if _, ok := entry.Extra[field]; ok {
				report.Items = append(report.Items, lockfile.LossItem{
					Field:        "packages." + path + "." + field,
					Reason:       "package metadata field preserved in lock encode but not in canonical graph",
					SourceFormat: format, Semantic: false, ProducerMajor: major, Category: "mapped",
				})
			}
		}
		for ek := range entry.Extra {
			found := false
			for _, f := range auditPackageExtraFields {
				if ek == f {
					found = true
					break
				}
			}
			if found {
				continue
			}
			report.Items = append(report.Items, lockfile.LossItem{
				Field:        "packages." + path + "." + ek,
				Reason:       "unknown package field not mapped to canonical graph",
				SourceFormat: format, Semantic: true, ProducerMajor: major, Category: "loss",
			})
		}
		if len(entry.BundledDependencies) > 0 {
			report.Items = append(report.Items, lockfile.LossItem{
				Field:        "packages." + path + ".bundledDependencies",
				Reason:       "bundledDependencies preserved in lock; bundle expansion deferred",
				SourceFormat: format, Semantic: false, ProducerMajor: major, Category: "mapped",
			})
		}
	}
	return report
}

func lossAgainstPrior(prior, out *Document) lockfile.LossReport {
	report := FieldLossAudit(prior)
	if prior == nil || out == nil {
		return report
	}
	for path, priorEntry := range prior.Packages {
		outEntry, ok := out.Packages[path]
		if !ok {
			report.Items = append(report.Items, lockfile.LossItem{
				Field: "packages." + path, Reason: "package path dropped on encode",
				SourceFormat: prior.Detection.Format, Semantic: true, ProducerMajor: prior.LockfileVersion, Category: "loss",
			})
			continue
		}
		for k, v := range priorEntry.Extra {
			if _, ok := outEntry.Extra[k]; !ok {
				report.Items = append(report.Items, lockfile.LossItem{
					Field: "packages." + path + "." + k, Reason: "package extra field lost on encode",
					Value: string(v), SourceFormat: prior.Detection.Format, Semantic: true,
					ProducerMajor: prior.LockfileVersion, Category: "loss",
				})
			}
		}
	}
	_ = report.Normalize()
	return report
}
