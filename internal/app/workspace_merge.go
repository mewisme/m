package app

import (
	"encoding/json"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/lockfile"
	"github.com/mewisme/m/internal/resolver"
)

func mergeFilteredWorkspaceResolution(
	prior *graph.Graph,
	priorExt lockfile.Extensions,
	active *resolver.Resolution,
	untouched []graph.ImporterID,
) (*resolver.Resolution, error) {
	if active == nil || prior == nil {
		return active, nil
	}
	activeIDs := importerIDsInGraph(active.Graph)
	removeKeys := map[string]struct{}{}
	for id := range activeIDs {
		if id == graph.RootImporter {
			continue
		}
		for k := range importerPackageClosure(prior, string(id)) {
			removeKeys[k] = struct{}{}
		}
	}

	b := graph.NewBuilder()
	importerSeen := map[graph.ImporterID]bool{}
	for _, im := range prior.Importers {
		b.Importer(im.ID, im.Name)
		importerSeen[im.ID] = true
	}
	for _, im := range active.Graph.Importers {
		if !importerSeen[im.ID] {
			b.Importer(im.ID, im.Name)
		}
	}

	activeKeys := map[string]struct{}{}
	for _, p := range active.Graph.Packages {
		activeKeys[p.ID.Key()] = struct{}{}
	}
	for _, p := range prior.Packages {
		key := p.ID.Key()
		if _, drop := removeKeys[key]; drop {
			continue
		}
		if _, exists := activeKeys[key]; exists {
			continue
		}
		b.Package(p.ID, p.Integrity, p.TarballURL)
	}
	for _, p := range active.Graph.Packages {
		b.Package(p.ID, p.Integrity, p.TarballURL)
	}

	activeEdge := map[string]struct{}{}
	for _, e := range active.Graph.Edges {
		b.EdgeEx(e.From, e.Name, e.To, e.Kind, e.Range, e.Optional)
		activeEdge[edgeKey(e)] = struct{}{}
	}
	for _, e := range prior.Edges {
		if _, ok := activeEdge[edgeKey(e)]; ok {
			continue
		}
		if _, drop := removeKeys[e.To]; drop {
			continue
		}
		if _, drop := removeKeys[e.From]; drop {
			continue
		}
		if activeIDs[graph.ImporterID(e.From)] {
			continue
		}
		b.EdgeEx(e.From, e.Name, e.To, e.Kind, e.Range, e.Optional)
	}

	g, err := b.Build()
	if err != nil {
		return nil, err
	}
	if err := validateMergedGraph(g); err != nil {
		return nil, err
	}
	ext, err := mergePriorExtensions(active.Extensions, priorExt, removeKeys, activeKeys)
	if err != nil {
		return nil, err
	}
	return &resolver.Resolution{
		SchemaVersion: active.SchemaVersion,
		Graph:         g,
		Decisions:     active.Decisions,
		Extensions:    ext,
	}, nil
}

func importerPackageClosure(g *graph.Graph, importerID string) map[string]struct{} {
	out := map[string]struct{}{}
	if g == nil {
		return out
	}
	nodes := map[string]struct{}{importerID: {}, string(graph.RootImporter): {}}
	changed := true
	for changed {
		changed = false
		for _, e := range g.Edges {
			fromIn := false
			toIn := false
			if _, ok := nodes[e.From]; ok {
				fromIn = true
			}
			if _, ok := nodes[e.To]; ok {
				toIn = true
			}
			if fromIn {
				if _, ok := nodes[e.To]; !ok {
					nodes[e.To] = struct{}{}
					changed = true
				}
			}
			if toIn {
				if _, ok := nodes[e.From]; !ok {
					nodes[e.From] = struct{}{}
					changed = true
				}
			}
		}
	}
	for k := range nodes {
		if k == importerID || k == string(graph.RootImporter) {
			continue
		}
		out[k] = struct{}{}
	}
	return out
}

func edgeKey(e graph.Edge) string {
	return e.From + "\x00" + e.Name + "\x00" + e.To
}

func validateMergedGraph(g *graph.Graph) error {
	if g == nil {
		return nil
	}
	pkgs := map[string]struct{}{}
	for _, p := range g.Packages {
		pkgs[p.ID.Key()] = struct{}{}
	}
	for _, e := range g.Edges {
		if _, ok := pkgs[e.To]; !ok {
			return apperr.New(apperr.Lockfile, "app.install.merge", e.To, "dangling edge in merged graph")
		}
	}
	return nil
}

func mergePriorExtensions(active, prior lockfile.Extensions, removed, activeKeys map[string]struct{}) (lockfile.Extensions, error) {
	if len(prior) == 0 {
		return active, nil
	}
	out := lockfile.Extensions{}
	for k, v := range active {
		out[k] = v
	}
	priorLocals, err := resolver.DecodeLocalSources(prior)
	if err != nil {
		return nil, err
	}
	if len(priorLocals) == 0 {
		return out, nil
	}
	activeLocals, _ := resolver.DecodeLocalSources(active)
	if activeLocals == nil {
		activeLocals = map[string]resolver.LocalSource{}
	}
	for key, src := range priorLocals {
		if _, inActive := activeLocals[key]; inActive {
			continue
		}
		if _, drop := removed[key]; drop {
			continue
		}
		if _, inActive := activeKeys[key]; inActive {
			continue
		}
		activeLocals[key] = src
	}
	if len(activeLocals) == 0 {
		return out, nil
	}
	raw, err := json.Marshal(activeLocals)
	if err != nil {
		return nil, err
	}
	out[resolver.LocalExtensionKey] = raw
	return out, nil
}
