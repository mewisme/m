package app

import (
	"encoding/json"
	"sort"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/lockfile"
	"github.com/mewisme/mew/internal/resolver"
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
	var activeImporterIDs []graph.ImporterID
	for id := range activeIDs {
		if id == graph.RootImporter {
			continue
		}
		activeImporterIDs = append(activeImporterIDs, id)
	}
	sort.Slice(activeImporterIDs, func(i, j int) bool { return activeImporterIDs[i] < activeImporterIDs[j] })

	activeClosure := dependencyClosureUnion(prior, activeImporterIDs)
	preservedClosure := dependencyClosureUnion(prior, untouched)
	removeKeys := computeRemoveKeys(activeClosure, preservedClosure)

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
	if err := validatePriorGraphTopology(prior); err != nil {
		return nil, err
	}
	preservedEdges := preservedSubgraphEdges(prior, untouched)
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
		if _, ok := preservedEdges[edgeKey(e)]; !ok {
			continue
		}
		b.EdgeEx(e.From, e.Name, e.To, e.Kind, e.Range, e.Optional)
	}

	g, err := b.Build()
	if err != nil {
		return nil, err
	}
	if err := validateMergedWorkspaceGraph(g, prior, untouched, nil); err != nil {
		return nil, err
	}
	ext, err := mergePriorExtensions(active.Extensions, priorExt, removeKeys, activeKeys)
	if err != nil {
		return nil, err
	}
	if err := validateMergedWorkspaceGraph(g, prior, untouched, ext); err != nil {
		return nil, err
	}
	return &resolver.Resolution{
		SchemaVersion: active.SchemaVersion,
		Graph:         g,
		Decisions:     active.Decisions,
		Extensions:    ext,
	}, nil
}

func dependencyClosure(g *graph.Graph, importerID graph.ImporterID) map[string]struct{} {
	return dependencyClosureUnion(g, []graph.ImporterID{importerID})
}

func dependencyClosureUnion(g *graph.Graph, importerIDs []graph.ImporterID) map[string]struct{} {
	out := map[string]struct{}{}
	if g == nil || len(importerIDs) == 0 {
		return out
	}
	importerKeys := map[string]struct{}{}
	for _, id := range importerIDs {
		importerKeys[string(id)] = struct{}{}
	}
	nodes := map[string]struct{}{string(graph.RootImporter): {}}
	for k := range importerKeys {
		nodes[k] = struct{}{}
	}
	changed := true
	for changed {
		changed = false
		for _, e := range g.Edges {
			if _, ok := nodes[e.From]; !ok {
				continue
			}
			if _, ok := nodes[e.To]; ok {
				continue
			}
			nodes[e.To] = struct{}{}
			changed = true
		}
	}
	for k := range nodes {
		if k == string(graph.RootImporter) {
			continue
		}
		if _, isImporter := importerKeys[k]; isImporter {
			continue
		}
		out[k] = struct{}{}
	}
	return out
}

func computeRemoveKeys(active, preserved map[string]struct{}) map[string]struct{} {
	out := map[string]struct{}{}
	for k := range active {
		if _, keep := preserved[k]; !keep {
			out[k] = struct{}{}
		}
	}
	return out
}

func edgeKey(e graph.Edge) string {
	return e.From + "\x00" + e.Name + "\x00" + e.To
}

func reachableFromImporter(g *graph.Graph, start graph.ImporterID) map[string]struct{} {
	nodes := map[string]struct{}{string(start): {}}
	changed := true
	for changed {
		changed = false
		for _, e := range g.Edges {
			if _, ok := nodes[e.From]; !ok {
				continue
			}
			if _, ok := nodes[e.To]; ok {
				continue
			}
			nodes[e.To] = struct{}{}
			changed = true
		}
	}
	return nodes
}

func preservedSubgraphEdges(prior *graph.Graph, untouched []graph.ImporterID) map[string]struct{} {
	out := map[string]struct{}{}
	if prior == nil {
		return out
	}
	for _, id := range untouched {
		reachable := reachableFromImporter(prior, id)
		for _, e := range prior.Edges {
			if _, ok := reachable[e.From]; !ok {
				continue
			}
			if _, ok := reachable[e.To]; !ok {
				continue
			}
			out[edgeKey(e)] = struct{}{}
		}
	}
	return out
}

func priorNodeKinds(g *graph.Graph) (pkgs, importers map[string]struct{}) {
	pkgs = map[string]struct{}{}
	importers = map[string]struct{}{}
	if g == nil {
		return pkgs, importers
	}
	for _, p := range g.Packages {
		pkgs[p.ID.Key()] = struct{}{}
	}
	for _, im := range g.Importers {
		importers[string(im.ID)] = struct{}{}
	}
	return pkgs, importers
}

func validatePriorGraphTopology(g *graph.Graph) error {
	if g == nil {
		return nil
	}
	pkgs, importers := priorNodeKinds(g)
	for _, e := range g.Edges {
		if _, ok := pkgs[e.To]; !ok {
			return apperr.New(apperr.Lockfile, "app.install.merge", e.To, "prior edge references missing package")
		}
		if _, ok := importers[e.From]; ok {
			continue
		}
		if _, ok := pkgs[e.From]; !ok {
			return apperr.New(apperr.Lockfile, "app.install.merge", e.From, "prior edge references missing source")
		}
	}
	return nil
}

func validateMergedWorkspaceGraph(g *graph.Graph, prior *graph.Graph, untouched []graph.ImporterID, ext lockfile.Extensions) error {
	if g == nil {
		return nil
	}
	pkgs := map[string]struct{}{}
	importers := map[string]struct{}{}
	for _, p := range g.Packages {
		pkgs[p.ID.Key()] = struct{}{}
	}
	for _, im := range g.Importers {
		importers[string(im.ID)] = struct{}{}
	}
	mergedEdges := map[string]graph.Edge{}
	for _, e := range g.Edges {
		if _, ok := pkgs[e.To]; !ok {
			return apperr.New(apperr.Lockfile, "app.install.merge", e.To, "dangling edge target in merged graph")
		}
		if _, ok := importers[e.From]; !ok {
			if _, ok := pkgs[e.From]; !ok {
				return apperr.New(apperr.Lockfile, "app.install.merge", e.From, "dangling edge source in merged graph")
			}
		}
		mergedEdges[edgeKey(e)] = e
	}
	for _, id := range untouched {
		for k := range dependencyClosure(prior, id) {
			if _, ok := pkgs[k]; !ok {
				return apperr.New(apperr.Lockfile, "app.install.merge", k, "missing package from untouched importer closure")
			}
		}
	}
	preserved := preservedSubgraphEdges(prior, untouched)
	for key := range preserved {
		merged, ok := mergedEdges[key]
		if !ok {
			return apperr.New(apperr.Lockfile, "app.install.merge", key, "missing preserved subgraph edge in merged graph")
		}
		for _, e := range prior.Edges {
			if edgeKey(e) != key {
				continue
			}
			if e.Name != merged.Name || e.Range != merged.Range || e.Kind != merged.Kind || e.Optional != merged.Optional {
				return apperr.New(apperr.Lockfile, "app.install.merge", key, "preserved subgraph edge metadata mismatch")
			}
			break
		}
	}
	for _, im := range g.Importers {
		if im.ID == graph.RootImporter {
			continue
		}
		for _, e := range g.Edges {
			if e.From != string(im.ID) {
				continue
			}
			if _, ok := pkgs[e.To]; !ok {
				return apperr.New(apperr.Lockfile, "app.install.merge", e.To, "importer dependency missing package")
			}
		}
	}
	if len(ext) == 0 {
		return nil
	}
	locals, err := resolver.DecodeLocalSources(ext)
	if err != nil {
		return err
	}
	for key := range locals {
		if _, ok := pkgs[key]; !ok {
			return apperr.New(apperr.Lockfile, "app.install.merge", key, "extension references missing package")
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
	activeLocals, err := resolver.DecodeLocalSources(active)
	if err != nil {
		return nil, err
	}
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
