package graph

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
)

// Validate normalizes ordering, checks identity uniqueness, and verifies edges.
// On success the graph is considered frozen for consumers.
func (g *Graph) Validate() error {
	if g == nil {
		return apperr.New(apperr.Lockfile, "graph.validate", "graph", "nil graph")
	}
	if g.SchemaVersion == 0 {
		g.SchemaVersion = SchemaVersion
	}
	if g.SchemaVersion < SchemaVersion {
		g.Migrate()
	}
	if g.SchemaVersion != SchemaVersion {
		return apperr.New(apperr.Lockfile, "graph.validate", "graph",
			fmt.Sprintf("unsupported schemaVersion %d", g.SchemaVersion))
	}

	for i := range g.Edges {
		NormalizeEdge(&g.Edges[i])
	}

	for i := range g.Packages {
		g.Packages[i].ID.Normalize()
	}

	sort.SliceStable(g.Importers, func(i, j int) bool {
		return string(g.Importers[i].ID) < string(g.Importers[j].ID)
	})
	sort.SliceStable(g.Packages, func(i, j int) bool {
		return g.Packages[i].ID.Key() < g.Packages[j].ID.Key()
	})
	sort.SliceStable(g.Edges, func(i, j int) bool {
		a, b := g.Edges[i], g.Edges[j]
		if a.From != b.From {
			return a.From < b.From
		}
		if a.Name != b.Name {
			return a.Name < b.Name
		}
		if a.To != b.To {
			return a.To < b.To
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		return a.Range < b.Range
	})

	importerSeen := make(map[ImporterID]struct{}, len(g.Importers))
	for _, im := range g.Importers {
		if im.ID == "" {
			return apperr.New(apperr.Lockfile, "graph.validate", "graph", "empty importer id")
		}
		if _, ok := importerSeen[im.ID]; ok {
			return apperr.New(apperr.Lockfile, "graph.validate", "graph",
				fmt.Sprintf("duplicate importer %q", im.ID))
		}
		importerSeen[im.ID] = struct{}{}
	}

	pkgByKey := make(map[string]Package, len(g.Packages))
	for _, p := range g.Packages {
		if p.ID.Name == "" || p.ID.Version == "" {
			return apperr.New(apperr.Lockfile, "graph.validate", "graph", "package missing name or version")
		}
		key := p.ID.Key()
		if prev, ok := pkgByKey[key]; ok {
			if prev.Integrity != p.Integrity || prev.TarballURL != p.TarballURL {
				return apperr.New(apperr.Lockfile, "graph.validate", "peer-context",
					fmt.Sprintf("peer-context identity collision for %q", key))
			}
			return apperr.New(apperr.Lockfile, "graph.validate", "peer-context",
				fmt.Sprintf("duplicate package key %q", key))
		}
		pkgByKey[key] = p
	}

	for _, e := range g.Edges {
		if e.Name == "" {
			return apperr.New(apperr.Lockfile, "graph.validate", "graph", "edge missing name")
		}
		if e.To == "" {
			return apperr.New(apperr.Lockfile, "graph.validate", "graph", "edge missing to")
		}
		if _, ok := pkgByKey[e.To]; !ok && !isLocalEdgeTarget(e.To) {
			return apperr.New(apperr.Lockfile, "graph.validate", "graph",
				fmt.Sprintf("dangling edge to %q", e.To))
		}
		if e.From == "" {
			return apperr.New(apperr.Lockfile, "graph.validate", "graph", "edge missing from")
		}
		if _, ok := pkgByKey[e.From]; !ok {
			if _, ok := importerSeen[ImporterID(e.From)]; !ok {
				return apperr.New(apperr.Lockfile, "graph.validate", "graph",
					fmt.Sprintf("edge from unknown %q", e.From))
			}
		}
	}
	return nil
}

func isLocalEdgeTarget(to string) bool {
	return strings.HasPrefix(to, "link:") ||
		strings.HasPrefix(to, "workspace:") ||
		strings.HasPrefix(to, "file:")
}
