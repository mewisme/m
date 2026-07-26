package mlock

import (
	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/manifest"
	"strings"
)

// ToGraph extracts the canonical graph from a lock document.
func ToGraph(d *Document) (*graph.Graph, error) {
	if d == nil {
		return nil, apperr.New(apperr.Lockfile, "mlock.graph", "m.lock", "nil document")
	}
	importers := make([]graph.Importer, 0, len(d.Importers))
	for _, im := range d.Importers {
		importers = append(importers, graph.Importer{
			ID:   im.ID,
			Name: im.Name,
			Path: im.Path,
		})
	}
	g := &graph.Graph{
		SchemaVersion: graph.SchemaVersion,
		Importers:     importers,
		Packages:      append([]graph.Package(nil), d.Packages...),
		Edges:         append([]graph.Edge(nil), d.Edges...),
	}
	if err := g.Validate(); err != nil {
		return nil, err
	}
	return g, nil
}

// FromGraph builds a lock document from a validated graph and importer specifiers.
func FromGraph(g *graph.Graph, specifiers map[graph.ImporterID][]Specifier, settings Settings) (*Document, error) {
	if g == nil {
		return nil, apperr.New(apperr.Lockfile, "mlock.graph", "graph", "nil graph")
	}
	if err := g.Validate(); err != nil {
		return nil, err
	}
	if err := settings.Normalize(); err != nil {
		return nil, err
	}

	importers := make([]ImporterSection, 0, len(g.Importers))
	for _, im := range g.Importers {
		specs := specifiers[im.ID]
		if specs == nil {
			specs = []Specifier{}
		}
		copied := append([]Specifier(nil), specs...)
		section := ImporterSection{
			ID:         im.ID,
			Name:       im.Name,
			Path:       im.Path,
			Specifiers: copied,
		}
		if err := normalizeSpecifiers(&section); err != nil {
			return nil, err
		}
		importers = append(importers, section)
	}

	doc := &Document{
		LockfileVersion: LockfileVersion,
		Settings:        settings,
		Importers:       importers,
		Packages:        append([]graph.Package(nil), g.Packages...),
		Edges:           append([]graph.Edge(nil), g.Edges...),
	}
	if err := doc.Normalize(); err != nil {
		return nil, err
	}
	return doc, nil
}

// SpecifiersFromManifest converts a normalized manifest to root specifiers.
func SpecifiersFromManifest(m *manifest.Manifest) []Specifier {
	if m == nil {
		return []Specifier{}
	}
	out := make([]Specifier, 0, len(m.Dependencies))
	for _, d := range m.Dependencies {
		out = append(out, Specifier{Name: d.Name, Range: d.Range, Kind: d.Kind})
	}
	return out
}

// SpecifiersFromGraph derives importer specifiers from importer edges.
func SpecifiersFromGraph(g *graph.Graph) map[graph.ImporterID][]Specifier {
	if g == nil {
		return nil
	}
	importerIDs := make(map[graph.ImporterID]struct{}, len(g.Importers))
	for _, im := range g.Importers {
		importerIDs[im.ID] = struct{}{}
	}
	out := make(map[graph.ImporterID][]Specifier)
	for _, e := range g.Edges {
		id := graph.ImporterID(e.From)
		if _, ok := importerIDs[id]; !ok {
			continue
		}
		out[id] = append(out[id], Specifier{Name: edgeDepName(e), Range: e.Range, Kind: e.Kind})
	}
	for id, specs := range out {
		section := ImporterSection{ID: id, Specifiers: specs}
		_ = normalizeSpecifiers(&section)
		out[id] = section.Specifiers
	}
	return out
}

// edgeDepName extracts the package name from an edge's To key (name@version).
func edgeDepName(e graph.Edge) string {
	to := e.To
	if i := strings.IndexByte(to, '#'); i >= 0 {
		to = to[:i]
	}
	if i := strings.LastIndexByte(to, '@'); i > 0 {
		return to[:i]
	}
	return to
}
