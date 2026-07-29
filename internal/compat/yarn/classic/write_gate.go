package classic

import (
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/lockfile"
)

// WriteGate blocks graph-changing yarn classic lock mutations.
func WriteGate(priorGraph, nextGraph *graph.Graph, prior []byte) error {
	same, err := lockfile.GraphsEqual(priorGraph, nextGraph)
	if err != nil {
		return err
	}
	if same {
		return nil
	}
	return lockfile.NewUnrepresentable("yarn.classic.write", "yarn.lock",
		"graph-changing yarn classic lock mutation is not supported until Yarn immutable CI certifies write",
		FieldLossAudit(nil))
}

// FieldLossAudit reports yarn classic fields not mapped into the canonical graph.
func FieldLossAudit(doc *Document) lockfile.LossReport {
	report := lockfile.LossReport{SchemaVersion: lockfile.LossReportSchemaVersion, Items: []lockfile.LossItem{}}
	if doc == nil {
		return report
	}
	for desc, blk := range doc.Blocks {
		for k := range blk.Extra {
			report.Items = append(report.Items, lockfile.LossItem{
				Field:        desc + "." + k,
				Reason:       "block field not mapped to canonical graph",
				SourceFormat: FormatClassic, Semantic: true, ProducerMajor: 1, Category: "loss",
			})
		}
	}
	return report
}
