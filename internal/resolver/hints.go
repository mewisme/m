package resolver

import (
	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/semver"
)

// graphHints wraps an optional partial graph for lock-prep selection.
type graphHints struct {
	g *graph.Graph

	incremental      bool
	updateClosure    map[string]struct{}
	priorSpecs       map[string]string
	manifestSpecs    map[string]string
	priorOverrides   map[string]string
	currentOverrides map[string]string
}

func (h graphHints) canPin(name string) bool {
	if h.g == nil {
		return false
	}
	if !h.incremental {
		return true
	}
	if _, inClosure := h.updateClosure[name]; inClosure {
		return false
	}
	if !mapsEqual(h.currentOverrides, h.priorOverrides) {
		return false
	}
	if priorRng, ok := h.priorSpecs[name]; ok {
		if curRng, ok := h.manifestSpecs[name]; ok && priorRng != curRng {
			return false
		}
	}
	return true
}

func (h graphHints) version(name, rng string) string {
	if h.g == nil || !h.canPin(name) {
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
	if h.g == nil || !h.canPin(name) {
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
