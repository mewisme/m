package berry

import (
	"sort"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/graph"
)

// ToGraph converts a Yarn Berry node-modules lock document to the canonical graph.
func ToGraph(doc *Document) (*graph.Graph, error) {
	if doc == nil {
		return nil, apperr.New(apperr.Lockfile, "yarn.berry.graph", "document", "nil document")
	}
	g := &graph.Graph{SchemaVersion: graph.SchemaVersion}
	pkgSeen := map[string]graph.Package{}

	for _, key := range sortedBlockKeys(doc.Blocks) {
		blk := doc.Blocks[key]
		if strings.Contains(key, "@workspace:") {
			im := graph.Importer{ID: graph.RootImporter, Name: workspaceName(key), Path: "."}
			g.Importers = append(g.Importers, im)
			continue
		}
		name, version, err := ParseResolution(blk.Resolution, blk.Version)
		if err != nil || version == "" {
			continue
		}
		pkgKey := PackageKey(name, version)
		pkgSeen[pkgKey] = graph.Package{ID: graph.PackageID{Name: name, Version: version}}
	}

	for _, key := range sortedKeys(pkgSeen) {
		g.Packages = append(g.Packages, pkgSeen[key])
	}

	for _, key := range sortedBlockKeys(doc.Blocks) {
		blk := doc.Blocks[key]
		if strings.Contains(key, "@workspace:") {
			from := string(graph.RootImporter)
			for depName, rng := range blk.Dependencies {
				to, ok := resolveBerryDep(doc, depName, rng)
				if !ok {
					return nil, apperr.New(apperr.Lockfile, "yarn.berry.graph", depName, "dangling workspace dependency")
				}
				g.Edges = append(g.Edges, graph.Edge{From: from, Name: depName, To: to, Kind: graph.DepProd, Range: rng})
			}
			continue
		}
		name, version, err := ParseResolution(blk.Resolution, blk.Version)
		if err != nil || version == "" {
			continue
		}
		from := PackageKey(name, version)
		for depName, rng := range blk.Dependencies {
			to, ok := resolveBerryDep(doc, depName, rng)
			if !ok {
				return nil, apperr.New(apperr.Lockfile, "yarn.berry.graph", depName, "dangling package dependency")
			}
			g.Edges = append(g.Edges, graph.Edge{From: from, Name: depName, To: to, Kind: graph.DepProd, Range: rng})
		}
	}

	if len(g.Importers) == 0 {
		g.Importers = append(g.Importers, graph.Importer{ID: graph.RootImporter, Path: "."})
	}
	if err := g.Validate(); err != nil {
		return nil, err
	}
	return g, nil
}

func resolveBerryDep(doc *Document, depName, rng string) (string, bool) {
	for key, blk := range doc.Blocks {
		if strings.Contains(key, depName) && (strings.Contains(key, rng) || strings.Contains(blk.Resolution, depName)) {
			name, version, err := ParseResolution(blk.Resolution, blk.Version)
			if err == nil && version != "" {
				return PackageKey(name, version), true
			}
		}
	}
	return "", false
}

func workspaceName(key string) string {
	if i := strings.Index(key, "@workspace:"); i >= 0 {
		return key[:i]
	}
	return key
}

// ParseResolution extracts name and version from a berry resolution string.
func ParseResolution(resolution, version string) (name, ver string, err error) {
	resolution = strings.TrimSpace(resolution)
	if resolution == "" {
		if version != "" {
			return "", version, nil
		}
		return "", "", apperr.New(apperr.Lockfile, "yarn.berry.identity", resolution, "empty resolution")
	}
	// resolution like "lodash@npm:4.18.1"
	base := resolution
	if i := strings.Index(base, ","); i >= 0 {
		base = strings.TrimSpace(base[:i])
	}
	at := strings.LastIndex(base, "@")
	if at <= 0 {
		return "", "", apperr.New(apperr.Lockfile, "yarn.berry.identity", resolution, "malformed resolution")
	}
	namePart := base[:at]
	verPart := strings.TrimPrefix(base[at+1:], "npm:")
	if strings.HasPrefix(verPart, "workspace:") {
		return namePart, verPart, nil
	}
	return namePart, verPart, nil
}

func sortedBlockKeys(m map[string]Block) []string {
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
