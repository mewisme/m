package resolver

import (
	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/manifest"
)

// BuildUpdateClosure returns package names that must be re-resolved for an update run.
func BuildUpdateClosure(targets []string, prior *graph.Graph, m *manifest.Manifest) map[string]struct{} {
	return buildUpdateClosure(targets, prior, m)
}

// buildUpdateClosure returns package names that must be re-resolved (not pinned from prior).
// Seeds are UpdateTargets, or all direct manifest dependencies when targets is empty.
// The closure includes each seed and packages reachable from prior lock edges below them.
func buildUpdateClosure(targets []string, prior *graph.Graph, m *manifest.Manifest) map[string]struct{} {
	closure := map[string]struct{}{}
	if prior == nil || m == nil {
		return closure
	}
	seeds := map[string]struct{}{}
	if len(targets) == 0 {
		for _, d := range m.Dependencies {
			if d.Kind == manifest.DepPeer {
				continue
			}
			seeds[d.Name] = struct{}{}
		}
	} else {
		for _, t := range targets {
			seeds[t] = struct{}{}
		}
	}
	for name := range seeds {
		closure[name] = struct{}{}
	}

	importers := importerIDs(prior)
	adj := make(map[string][]string, len(prior.Edges))
	for _, e := range prior.Edges {
		adj[e.From] = append(adj[e.From], e.To)
	}

	seen := map[string]struct{}{}
	var queue []string
	for _, p := range prior.Packages {
		if _, ok := seeds[p.ID.Name]; !ok {
			continue
		}
		key := p.ID.Key()
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		queue = append(queue, key)
	}
	// Also walk from importer edges into seeded direct dependencies.
	for _, e := range prior.Edges {
		if _, ok := importers[graph.ImporterID(e.From)]; !ok {
			continue
		}
		id := parsePackageKey(e.To)
		if _, ok := seeds[id.Name]; !ok {
			continue
		}
		if _, ok := seen[e.To]; ok {
			continue
		}
		seen[e.To] = struct{}{}
		queue = append(queue, e.To)
	}

	for len(queue) > 0 {
		key := queue[0]
		queue = queue[1:]
		id := parsePackageKey(key)
		closure[id.Name] = struct{}{}
		for _, to := range adj[key] {
			if _, ok := seen[to]; ok {
				continue
			}
			seen[to] = struct{}{}
			queue = append(queue, to)
		}
	}
	return closure
}

func importerIDs(g *graph.Graph) map[graph.ImporterID]struct{} {
	out := make(map[graph.ImporterID]struct{}, len(g.Importers))
	for _, im := range g.Importers {
		out[im.ID] = struct{}{}
	}
	return out
}

func rootImporterSpecifiers(g *graph.Graph) map[string]string {
	out := map[string]string{}
	if g == nil {
		return out
	}
	importers := importerIDs(g)
	for _, e := range g.Edges {
		if _, ok := importers[graph.ImporterID(e.From)]; !ok {
			continue
		}
		if e.From != string(graph.RootImporter) {
			continue
		}
		id := parsePackageKey(e.To)
		out[id.Name] = e.Range
	}
	return out
}

func manifestSpecifierMap(m *manifest.Manifest) map[string]string {
	out := map[string]string{}
	if m == nil {
		return out
	}
	for _, d := range m.Dependencies {
		out[d.Name] = d.Range
	}
	return out
}

func mapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func prepareHints(opts ResolveOptions, m *manifest.Manifest) graphHints {
	h := graphHints{g: opts.Hints}
	if opts.Prior != nil {
		if h.g == nil {
			h.g = opts.Prior
		}
		if opts.IncrementalUpdate {
			h.incremental = true
			h.updateClosure = buildUpdateClosure(opts.UpdateTargets, opts.Prior, m)
			h.priorSpecs = rootImporterSpecifiers(opts.Prior)
			h.manifestSpecs = manifestSpecifierMap(m)
			h.priorOverrides = opts.PriorOverrides
			h.currentOverrides = m.Overrides
		}
	}
	return h
}
