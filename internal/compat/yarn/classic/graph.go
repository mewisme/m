package classic

import (
	"sort"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/graph"
)

// ToGraph converts a Yarn classic lock document to the canonical graph.
func ToGraph(doc *Document) (*graph.Graph, error) {
	if doc == nil {
		return nil, apperr.New(apperr.Lockfile, "yarn.classic.graph", "document", "nil document")
	}
	g := &graph.Graph{
		SchemaVersion: graph.SchemaVersion,
		Importers:     []graph.Importer{{ID: graph.RootImporter, Path: "."}},
	}
	pkgSeen := map[string]graph.Package{}
	for _, desc := range sortedDescriptors(doc.Blocks) {
		blk := doc.Blocks[desc]
		name := DescriptorName(desc)
		pkgName, version := ParseVersion(name, blk.Version)
		if version == "" {
			continue
		}
		key := PackageKey(pkgName, version)
		pkgSeen[key] = graph.Package{
			ID:         graph.PackageID{Name: pkgName, Version: version},
			Integrity:  blk.Integrity,
			TarballURL: blk.Resolved,
		}
	}
	for _, key := range sortedKeys(pkgSeen) {
		g.Packages = append(g.Packages, pkgSeen[key])
	}
	for _, desc := range sortedDescriptors(doc.Blocks) {
		blk := doc.Blocks[desc]
		name := DescriptorName(desc)
		pkgName, version := ParseVersion(name, blk.Version)
		if version == "" {
			continue
		}
		from := PackageKey(pkgName, version)
		for depName, rng := range blk.Dependencies {
			to, ok := resolveDep(doc, depName, rng)
			if !ok {
				return nil, apperr.New(apperr.Lockfile, "yarn.classic.graph", depName, "dangling dependency")
			}
			g.Edges = append(g.Edges, graph.Edge{From: from, Name: depName, To: to, Kind: graph.DepProd, Range: rng})
		}
		// Root importer edges for top-level descriptors (no @ in parent chain).
		if !strings.Contains(desc, ", ") {
			rootTo := PackageKey(pkgName, version)
			if _, exists := pkgSeen[rootTo]; exists {
				g.Edges = append(g.Edges, graph.Edge{
					From: string(graph.RootImporter), Name: pkgName, To: rootTo, Kind: graph.DepProd, Range: descriptorRange(desc),
				})
			}
		}
	}
	if err := g.Validate(); err != nil {
		return nil, err
	}
	return g, nil
}

func resolveDep(doc *Document, depName, rng string) (string, bool) {
	for desc, blk := range doc.Blocks {
		if DescriptorName(desc) != depName {
			continue
		}
		if strings.Contains(desc, rng) || descriptorRange(desc) == rng {
			name := DescriptorName(desc)
			pkgName, version := ParseVersion(name, blk.Version)
			if version != "" {
				return PackageKey(pkgName, version), true
			}
		}
	}
	for desc, blk := range doc.Blocks {
		if DescriptorName(desc) == depName {
			name := DescriptorName(desc)
			pkgName, version := ParseVersion(name, blk.Version)
			if version != "" {
				return PackageKey(pkgName, version), true
			}
		}
	}
	return "", false
}

func descriptorRange(desc string) string {
	at := strings.LastIndex(desc, "@")
	if at < 0 {
		return ""
	}
	return desc[at+1:]
}

func sortedDescriptors(m map[string]Block) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedKeys(m map[string]graph.Package) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// PackageKey returns the canonical graph key for name@version.
func PackageKey(name, version string) string {
	return name + "@" + version
}
