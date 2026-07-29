package berry

import (
	"encoding/json"

	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/lockfile"
)

// ApplyPnPTag marks a graph as PnP-linked for install gating.
func ApplyPnPTag(g *graph.Graph, ext lockfile.Extensions) (lockfile.Extensions, error) {
	if ext == nil {
		ext = lockfile.Extensions{}
	}
	raw, err := json.Marshal("pnp")
	if err != nil {
		return ext, err
	}
	ext[ExtLinkerKey] = raw
	if g != nil {
		_ = g
	}
	return ext, nil
}

// IsPnPGraph reports whether extensions mark a PnP linker.
func IsPnPGraph(ext lockfile.Extensions) bool {
	if ext == nil {
		return false
	}
	raw, ok := ext[ExtLinkerKey]
	if !ok {
		return false
	}
	var linker string
	if err := json.Unmarshal(raw, &linker); err != nil {
		return false
	}
	return linker == "pnp"
}

// ToPnPGraph parses a berry lock for validate/migrate with PnP linker tag.
func ToPnPGraph(doc *Document) (*graph.Graph, lockfile.Extensions, error) {
	g, err := ToGraph(doc)
	if err != nil {
		return nil, nil, err
	}
	doc.Linker = "pnp"
	doc.Detection.Format = FormatBerryPnP
	ext, err := ApplyPnPTag(g, doc.Extensions)
	if err != nil {
		return nil, nil, err
	}
	return g, ext, nil
}

// FieldLossAudit reports berry fields not mapped into the canonical graph.
func FieldLossAudit(doc *Document) lockfile.LossReport {
	report := lockfile.LossReport{SchemaVersion: lockfile.LossReportSchemaVersion, Items: []lockfile.LossItem{}}
	if doc == nil {
		return report
	}
	format := doc.Detection.Format
	if format == "" {
		format = FormatBerryNM
	}
	for key, blk := range doc.Blocks {
		if blk.Checksum != "" {
			report.Items = append(report.Items, lockfile.LossItem{
				Field: key + ".checksum", Reason: "checksum preserved in lock but not in canonical graph",
				SourceFormat: format, Semantic: false, Category: "mapped",
			})
		}
		for ek := range blk.Extra {
			report.Items = append(report.Items, lockfile.LossItem{
				Field: key + "." + ek, Reason: "block field not mapped to canonical graph",
				SourceFormat: format, Semantic: true, Category: "loss",
			})
		}
	}
	return report
}
