package bun

import (
	"sort"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/graph"
)

// ToGraph converts a bun lock document to the canonical graph.
func ToGraph(doc *Document) (*graph.Graph, error) {
	if doc == nil {
		return nil, apperr.New(apperr.Lockfile, "bun.graph", "document", "nil document")
	}
	if err := ValidateSupported(doc); err != nil {
		return nil, err
	}
	g := &graph.Graph{SchemaVersion: graph.SchemaVersion}

	pkgKeys := map[string]graph.Package{}
	for shortName, arr := range doc.Packages {
		resolution, _, integrity, _, err := parsePackageTuple(arr)
		if err != nil {
			return nil, apperr.Wrap(apperr.Lockfile, "bun.graph", shortName, err)
		}
		name, version, err := ParseResolution(resolution)
		if err != nil {
			return nil, err
		}
		if version == "" {
			continue
		}
		key := PackageKey(name, version)
		pkgKeys[key] = graph.Package{
			ID:        graph.PackageID{Name: name, Version: version},
			Integrity: integrity,
		}
		_ = shortName
	}
	for _, key := range sortedKeys(pkgKeys) {
		g.Packages = append(g.Packages, pkgKeys[key])
	}

	wsPaths := sortedWorkspacePaths(doc.Workspaces)
	for _, path := range wsPaths {
		entry := doc.Workspaces[path]
		im := graph.Importer{
			ID:   importerIDForPath(path),
			Name: entry.Name,
			Path: importerPath(path),
		}
		g.Importers = append(g.Importers, im)
		from := string(im.ID)
		if err := appendWorkspaceEdges(g, doc, from, entry); err != nil {
			return nil, err
		}
	}

	for shortName, arr := range doc.Packages {
		resolution, _, _, info, err := parsePackageTuple(arr)
		if err != nil {
			return nil, err
		}
		name, version, err := ParseResolution(resolution)
		if err != nil || version == "" {
			continue
		}
		from := PackageKey(name, version)
		if err := appendInfoEdges(g, doc, from, info); err != nil {
			return nil, err
		}
		_ = shortName
	}

	if len(g.Importers) == 0 {
		g.Importers = append(g.Importers, graph.Importer{ID: graph.RootImporter, Path: "."})
	}
	if err := g.Validate(); err != nil {
		return nil, err
	}
	return g, nil
}

func appendWorkspaceEdges(g *graph.Graph, doc *Document, from string, entry WorkspaceEntry) error {
	fields := []struct {
		deps map[string]string
		kind graph.DepKind
		opt  bool
	}{
		{entry.Dependencies, graph.DepProd, false},
		{entry.DevDependencies, graph.DepDev, false},
		{entry.OptionalDependencies, graph.DepOptional, true},
	}
	for _, f := range fields {
		for depName, rng := range f.deps {
			to, ok := resolveShortName(doc, depName)
			if !ok {
				return apperr.New(apperr.Lockfile, "bun.graph", depName, "dangling workspace dependency")
			}
			g.Edges = append(g.Edges, graph.Edge{From: from, Name: depName, To: to, Kind: f.kind, Range: rng, Optional: f.opt})
		}
	}
	return nil
}

func appendInfoEdges(g *graph.Graph, doc *Document, from string, info PackageInfo) error {
	fields := []struct {
		deps map[string]string
		kind graph.DepKind
		opt  bool
	}{
		{info.Dependencies, graph.DepProd, false},
		{info.DevDependencies, graph.DepDev, false},
		{info.OptionalDependencies, graph.DepOptional, true},
		{info.PeerDependencies, graph.DepPeer, false},
	}
	for _, f := range fields {
		for depName, rng := range f.deps {
			to, ok := resolveShortName(doc, depName)
			if !ok {
				return apperr.New(apperr.Lockfile, "bun.graph", depName, "dangling package dependency")
			}
			g.Edges = append(g.Edges, graph.Edge{From: from, Name: depName, To: to, Kind: f.kind, Range: rng, Optional: f.opt})
		}
	}
	return nil
}

func resolveShortName(doc *Document, depName string) (string, bool) {
	arr, ok := doc.Packages[depName]
	if !ok {
		return "", false
	}
	resolution, _, _, _, err := parsePackageTuple(arr)
	if err != nil {
		return "", false
	}
	name, version, err := ParseResolution(resolution)
	if err != nil || version == "" {
		return "", false
	}
	return PackageKey(name, version), true
}

func sortedWorkspacePaths(m map[string]WorkspaceEntry) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	if len(out) == 0 {
		out = []string{""}
	}
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

func importerIDForPath(path string) graph.ImporterID {
	if path == "" {
		return graph.RootImporter
	}
	return graph.ImporterID(path)
}

func importerPath(path string) string {
	if path == "" {
		return "."
	}
	return path
}

// PackageKey returns the canonical graph key for name@version.
func PackageKey(name, version string) string {
	return name + "@" + version
}
