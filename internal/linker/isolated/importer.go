package isolated

import (
	"path/filepath"

	"github.com/mewisme/mew/internal/graph"
)

// PrivateNMForEdgeFrom returns the virtual-store private node_modules dir for an edge source.
// Edge From is usually an importer ID; workspace member importers map to their package key.
func PrivateNMForEdgeFrom(nmRoot string, g *graph.Graph, pkgs []pkgLayout, from string) string {
	if priv := privateNMFor(pkgs, from); priv != "" {
		return priv
	}
	if from == string(graph.RootImporter) {
		return ""
	}
	for _, im := range g.Importers {
		if string(im.ID) != from {
			continue
		}
		for _, p := range g.Packages {
			if p.ID.Name == im.Name {
				return privateNMFor(pkgs, p.ID.Key())
			}
		}
	}
	return ""
}

// DepLinkPath is the expected node_modules link path for a dependency edge under isolated layout.
func DepLinkPath(nmRoot string, g *graph.Graph, pkgs []pkgLayout, e graph.Edge) string {
	privateNM := PrivateNMForEdgeFrom(nmRoot, g, pkgs, e.From)
	if privateNM == "" {
		return ""
	}
	depName := e.Name
	if depName == "" {
		depName = graph.TargetNameFromKey(e.To)
	}
	return filepath.Join(append([]string{privateNM}, installSegments(depName)...)...)
}
