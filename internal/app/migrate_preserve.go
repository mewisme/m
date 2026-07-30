package app

import (
	"encoding/json"

	"github.com/mewisme/mew/internal/compat/pnpm"
	"github.com/mewisme/mew/internal/lockfile"
	"github.com/mewisme/mew/internal/project"
)

const (
	mewMigratePnpmExt = "mew.migrate/pnpm"
	mewMigrateNubExt  = "mew.migrate/nub"
)

// preservePnpmMigrateLoss packs unmapped pnpm/nub fields into an m.lock extension
// (migration policy B) and reclassifies those loss items as non-semantic.
func preservePnpmMigrateLoss(fromID project.Identity, prior []byte, loss *lockfile.LossReport) (lockfile.Extensions, error) {
	if loss == nil || len(loss.Items) == 0 {
		return nil, nil
	}
	extKey := ""
	switch fromID {
	case project.IdentityPNPM:
		extKey = mewMigratePnpmExt
	case project.IdentityNub:
		extKey = mewMigrateNubExt
	default:
		return nil, nil
	}
	doc, err := pnpm.Decode(prior)
	if err != nil {
		return nil, err
	}
	sidecar := buildPnpmMigrateSidecar(doc)
	if sidecarEmpty(sidecar) {
		return nil, nil
	}
	raw, err := json.Marshal(sidecar)
	if err != nil {
		return nil, err
	}
	for i := range loss.Items {
		if !loss.Items[i].Semantic {
			continue
		}
		loss.Items[i].Semantic = false
		loss.Items[i].Category = "extension"
		loss.Items[i].Reason = loss.Items[i].Reason + "; preserved in m.lock extensions." + extKey
	}
	return lockfile.Extensions{extKey: raw}, nil
}

type pnpmMigrateSidecar struct {
	Settings  map[string]any             `json:"settings,omitempty"`
	Packages  map[string]map[string]any  `json:"packages,omitempty"`
	Snapshots map[string]map[string]any  `json:"snapshots,omitempty"`
	Importers map[string]map[string]any  `json:"importers,omitempty"`
	TopLevel  map[string]json.RawMessage `json:"topLevel,omitempty"`
}

func buildPnpmMigrateSidecar(doc *pnpm.Document) pnpmMigrateSidecar {
	var out pnpmMigrateSidecar
	if doc == nil {
		return out
	}
	if len(doc.Settings) > 0 {
		out.Settings = doc.Settings
	}
	if len(doc.Extensions) > 0 {
		out.TopLevel = doc.Extensions
	}
	for key, p := range doc.Packages {
		meta := map[string]any{}
		if len(p.Engines) > 0 {
			meta["engines"] = p.Engines
		}
		if p.Checksum != "" {
			meta["checksum"] = p.Checksum
		}
		if p.BuildPolicy != nil {
			meta["buildPolicy"] = p.BuildPolicy
		}
		for ek, ev := range p.Extra {
			meta[ek] = ev
		}
		if len(meta) == 0 {
			continue
		}
		if out.Packages == nil {
			out.Packages = map[string]map[string]any{}
		}
		out.Packages[key] = meta
	}
	for key, snap := range doc.Snapshots {
		meta := stripSnapshotTopologyForMigrate(snap)
		if len(meta) == 0 {
			continue
		}
		if out.Snapshots == nil {
			out.Snapshots = map[string]map[string]any{}
		}
		out.Snapshots[key] = meta
	}
	for id, sec := range doc.Importers {
		meta := map[string]any{}
		if len(sec.DependenciesMeta) > 0 {
			meta["dependenciesMeta"] = sec.DependenciesMeta
		}
		if sec.PublishDirectory != "" {
			meta["publishDirectory"] = sec.PublishDirectory
		}
		for ek, ev := range sec.Extra {
			var v any
			if err := json.Unmarshal(ev, &v); err == nil {
				meta[ek] = v
			} else {
				meta[ek] = ev
			}
		}
		if len(meta) == 0 {
			continue
		}
		if out.Importers == nil {
			out.Importers = map[string]map[string]any{}
		}
		out.Importers[id] = meta
	}
	return out
}

func stripSnapshotTopologyForMigrate(snap map[string]any) map[string]any {
	if len(snap) == 0 {
		return nil
	}
	out := make(map[string]any, len(snap))
	for k, v := range snap {
		switch k {
		case "dependencies", "optionalDependencies", "peerDependencies":
			continue
		default:
			out[k] = v
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func sidecarEmpty(s pnpmMigrateSidecar) bool {
	return len(s.Settings) == 0 && len(s.Packages) == 0 && len(s.Snapshots) == 0 &&
		len(s.Importers) == 0 && len(s.TopLevel) == 0
}
