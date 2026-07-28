package pnpm

import (
	"bytes"
	"encoding/json"
	"sort"

	"github.com/mewisme/mew/internal/lockfile"
)

func lossAgainstPrior(prior, out *Document) lockfile.LossReport {
	report := lockfile.LossReport{SchemaVersion: lockfile.LossReportSchemaVersion, Items: []lockfile.LossItem{}}
	for k, v := range prior.Extensions {
		if _, ok := out.Extensions[k]; !ok {
			report.Items = append(report.Items, lockfile.LossItem{Field: k, Reason: "extension dropped", Value: string(v)})
		}
	}
	for k, snap := range prior.Snapshots {
		if len(snap) == 0 {
			continue
		}
		outSnap, ok := out.Snapshots[k]
		if !ok || !mapsEqual(snap, outSnap) {
			report.Items = append(report.Items, lockfile.LossItem{Field: "snapshots." + k, Reason: "snapshot metadata changed or dropped"})
		}
	}
	for k, p := range prior.Packages {
		o, ok := out.Packages[k]
		if !ok {
			continue
		}
		if p.Checksum != "" && p.Checksum != o.Checksum {
			report.Items = append(report.Items, lockfile.LossItem{Field: "packages." + k + ".checksum", Reason: "checksum would change"})
		}
		if p.BuildPolicy != nil && !anyEqual(p.BuildPolicy, o.BuildPolicy) {
			report.Items = append(report.Items, lockfile.LossItem{Field: "packages." + k + ".buildPolicy", Reason: "buildPolicy would change"})
		}
	}
	sort.SliceStable(report.Items, func(i, j int) bool { return report.Items[i].Field < report.Items[j].Field })
	return report
}

func mapsEqual(a, b map[string]any) bool {
	ja, _ := json.Marshal(a)
	jb, _ := json.Marshal(b)
	return bytes.Equal(ja, jb)
}

func anyEqual(a, b any) bool {
	ja, _ := json.Marshal(a)
	jb, _ := json.Marshal(b)
	return bytes.Equal(ja, jb)
}
