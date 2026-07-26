package resolver

import (
	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/semver"
)

// graphHints wraps an optional partial graph for lock-prep selection.
type graphHints struct {
	g *graph.Graph
}

func (h graphHints) version(name, rng string) string {
	if h.g == nil {
		return ""
	}
	for _, p := range h.g.Packages {
		if p.ID.Name != name {
			continue
		}
		ok, err := semver.Satisfies(p.ID.Version, rng)
		if err != nil || !ok {
			continue
		}
		return p.ID.Version
	}
	return ""
}

func (h graphHints) pkg(name, version string) (graph.Package, bool) {
	if h.g == nil {
		return graph.Package{}, false
	}
	key := graph.PackageID{Name: name, Version: version}.Key()
	for _, p := range h.g.Packages {
		if p.ID.Key() == key {
			return p, true
		}
	}
	return graph.Package{}, false
}
