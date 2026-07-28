package pnpm

import (
	"bytes"
	"encoding/json"
	"sort"

	"github.com/mewisme/mew/internal/lockfile"
)

const (
	snapFieldDependencies         = "dependencies"
	snapFieldOptionalDependencies = "optionalDependencies"
	snapFieldPeerDependencies     = "peerDependencies"
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
		if !ok {
			if snapHasOnlyTopology(snap) {
				continue
			}
			report.Items = append(report.Items, lockfile.LossItem{Field: "snapshots." + k, Reason: "snapshot metadata dropped"})
			continue
		}
		if !snapshotMetadataEqual(snap, outSnap) {
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
		for fk, fv := range p.Extra {
			ov, ok := o.Extra[fk]
			if !ok || !anyEqual(fv, ov) {
				report.Items = append(report.Items, lockfile.LossItem{Field: "packages." + k + "." + fk, Reason: "package extra field would change"})
			}
		}
	}
	for id, sec := range prior.Importers {
		outSec, ok := out.Importers[id]
		if !ok {
			continue
		}
		for k, v := range sec.Extra {
			if outSec.Extra == nil {
				report.Items = append(report.Items, lockfile.LossItem{Field: "importers." + id + "." + k, Reason: "importer extra dropped"})
				continue
			}
			if !bytes.Equal(outSec.Extra[k], v) {
				report.Items = append(report.Items, lockfile.LossItem{Field: "importers." + id + "." + k, Reason: "importer extra changed"})
			}
		}
		if sec.PublishDirectory != "" && sec.PublishDirectory != outSec.PublishDirectory {
			report.Items = append(report.Items, lockfile.LossItem{Field: "importers." + id + ".publishDirectory", Reason: "publishDirectory would change"})
		}
	}
	sort.SliceStable(report.Items, func(i, j int) bool { return report.Items[i].Field < report.Items[j].Field })
	return report
}

func snapshotMetadataEqual(a, b map[string]any) bool {
	return mapsEqual(stripSnapshotTopology(a), stripSnapshotTopology(b))
}

func snapHasOnlyTopology(snap map[string]any) bool {
	return len(stripSnapshotTopology(snap)) == 0
}

func stripSnapshotTopology(snap map[string]any) map[string]any {
	if snap == nil {
		return nil
	}
	out := make(map[string]any, len(snap))
	for k, v := range snap {
		switch k {
		case snapFieldDependencies, snapFieldOptionalDependencies, snapFieldPeerDependencies:
			continue
		default:
			out[k] = v
		}
	}
	return out
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
