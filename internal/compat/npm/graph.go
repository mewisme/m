package npm

import (
	"sort"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/lockfile"
)

// ToGraph converts an npm lock document to the canonical graph.
func ToGraph(doc *Document) (*graph.Graph, error) {
	if doc == nil {
		return nil, apperr.New(apperr.Lockfile, "npm.graph", "document", "nil document")
	}
	if err := ValidateSupported(doc); err != nil {
		return nil, err
	}
	idx := BuildIndex(doc.Packages)
	g := &graph.Graph{SchemaVersion: graph.SchemaVersion}

	importerPaths := importerPaths(doc)
	for _, path := range importerPaths {
		entry := doc.Packages[path]
		im := graph.Importer{
			ID:   importerIDForPath(path),
			Name: entry.Name,
			Path: path,
		}
		if im.Name == "" && path != "" && !strings.HasPrefix(path, "node_modules/") {
			if idx := strings.LastIndex(path, "/"); idx >= 0 {
				im.Name = path[idx+1:]
			} else {
				im.Name = path
			}
		}
		if im.Path == "" {
			im.Path = "."
		}
		g.Importers = append(g.Importers, im)
	}

	pkgSeen := map[string]graph.Package{}
	for path, entry := range doc.Packages {
		if path == "" || entry.Link {
			continue
		}
		key, ok := entryPackageKey(path, entry)
		if !ok {
			continue
		}
		name, version, err := ParsePackageKey(key)
		if err != nil {
			return nil, err
		}
		pkg := graph.Package{
			ID:         graph.PackageID{Name: name, Version: version},
			Integrity:  entry.Integrity,
			TarballURL: entry.Resolved,
		}
		if entry.Link {
			continue
		}
		pkgSeen[key] = pkg
	}
	for _, key := range sortedKeys(pkgSeen) {
		g.Packages = append(g.Packages, pkgSeen[key])
	}

	for _, path := range importerPaths {
		entry := doc.Packages[path]
		from := string(importerIDForPath(path))
		if err := appendDepEdges(g, doc.Packages, idx, from, path, entry); err != nil {
			return nil, err
		}
	}

	for path, entry := range doc.Packages {
		if isWorkspaceImporter(path, entry) || entry.Link {
			continue
		}
		key, ok := entryPackageKey(path, entry)
		if !ok {
			continue
		}
		if err := appendDepEdges(g, doc.Packages, idx, key, path, entry); err != nil {
			return nil, err
		}
	}

	if err := g.Validate(); err != nil {
		return nil, err
	}
	return g, nil
}

func importerPaths(doc *Document) []string {
	var paths []string
	for path, entry := range doc.Packages {
		if isWorkspaceImporter(path, entry) {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		paths = []string{""}
	}
	return paths
}

func appendDepEdges(g *graph.Graph, packages map[string]PackageEntry, idx *PackageIndex, from, path string, entry PackageEntry) error {
	type depField struct {
		deps map[string]string
		kind graph.DepKind
		opt  bool
	}
	fields := []depField{
		{entry.Dependencies, graph.DepProd, false},
		{entry.DevDependencies, graph.DepDev, false},
		{entry.OptionalDependencies, graph.DepOptional, true},
		{entry.PeerDependencies, graph.DepPeer, false},
	}
	for _, field := range fields {
		if len(field.deps) == 0 {
			continue
		}
		names := sortedStringKeys(field.deps)
		for _, depName := range names {
			if err := validateDepName(depName); err != nil {
				return err
			}
			spec := field.deps[depName]
			targetKey, err := idx.ResolveDep(packages, path, depName)
			if err != nil {
				return err
			}
			g.Edges = append(g.Edges, graph.Edge{
				From:     from,
				Name:     depName,
				To:       targetKey,
				Kind:     field.kind,
				Range:    spec,
				Optional: field.opt,
			})
		}
	}
	return nil
}

// FromGraph builds an npm lock document from a graph and optional prior template.
func FromGraph(g *graph.Graph, prior *Document, det lockfile.Detection) (*Document, error) {
	if g == nil {
		return nil, apperr.New(apperr.Lockfile, "npm.graph", "graph", "nil graph")
	}
	if err := g.Validate(); err != nil {
		return nil, err
	}
	doc := &Document{
		Requires:  true,
		Packages:  map[string]PackageEntry{},
		Detection: det,
	}
	if prior != nil {
		cloned := prior.Clone()
		doc.LockfileVersion = cloned.LockfileVersion
		doc.Name = cloned.Name
		doc.Extensions = cloned.Extensions
	}
	if doc.LockfileVersion == 0 {
		doc.LockfileVersion = defaultLockfileVersion(det)
	}

	pathByKey := map[string]string{}
	if prior != nil {
		for path, entry := range prior.Packages {
			if key, ok := entryPackageKey(path, entry); ok {
				pathByKey[key] = path
			}
		}
	}

	for _, im := range g.Importers {
		path := importerPath(im)
		entry := doc.Packages[path]
		if entry.Name == "" {
			entry.Name = im.Name
		}
		if path == "" && entry.Version == "" {
			if v := rootVersion(g); v != "" {
				entry.Version = v
			}
		}
		doc.Packages[path] = entry
	}

	for _, pkg := range g.Packages {
		key := pkg.ID.Key()
		path := pathByKey[key]
		if path == "" {
			path = hoistedPath(pkg.ID.Name)
		}
		entry := doc.Packages[path]
		entry.Name = pkg.ID.Name
		entry.Version = pkg.ID.Version
		entry.Integrity = pkg.Integrity
		entry.Resolved = pkg.TarballURL
		doc.Packages[path] = entry
		pathByKey[key] = path
	}

	edgeFields := edgesByFrom(g)
	for from, fields := range edgeFields {
		path := from
		if from == string(graph.RootImporter) {
			path = ""
		} else if p, ok := pathByKey[from]; ok {
			path = p
		}
		entry := doc.Packages[path]
		for kind, deps := range fields {
			applyDepField(&entry, kind, deps)
		}
		doc.Packages[path] = entry
	}

	if prior != nil {
		preservePriorPackageFields(doc, prior)
		restoreWorkspaceLinks(doc, prior)
	}
	return doc, nil
}

func restoreWorkspaceLinks(doc, prior *Document) {
	if doc == nil || prior == nil {
		return
	}
	for path, entry := range prior.Packages {
		if !entry.Link {
			continue
		}
		if _, ok := doc.Packages[path]; !ok {
			doc.Packages[path] = entry.clone()
		}
	}
}

func defaultLockfileVersion(det lockfile.Detection) int {
	switch det.Format {
	case FormatV2:
		return 2
	default:
		return 3
	}
}

func importerPath(im graph.Importer) string {
	if im.ID == graph.RootImporter || im.Path == "." {
		return ""
	}
	if im.Path != "" {
		return im.Path
	}
	return string(im.ID)
}

func rootVersion(g *graph.Graph) string {
	for _, im := range g.Importers {
		if im.ID == graph.RootImporter {
			return ""
		}
	}
	return ""
}

func hoistedPath(name string) string {
	return "node_modules/" + name
}

type depFields map[graph.DepKind]map[string]string

func edgesByFrom(g *graph.Graph) map[string]depFields {
	out := map[string]depFields{}
	for _, e := range g.Edges {
		if out[e.From] == nil {
			out[e.From] = depFields{}
		}
		if out[e.From][e.Kind] == nil {
			out[e.From][e.Kind] = map[string]string{}
		}
		if e.Range != "" {
			out[e.From][e.Kind][e.Name] = e.Range
		} else if ver := graphPackageVersion(e.To); ver != "" {
			out[e.From][e.Kind][e.Name] = "^" + ver
		}
	}
	return out
}

func graphPackageVersion(key string) string {
	_, ver, err := ParsePackageKey(key)
	if err != nil {
		return ""
	}
	return ver
}

func applyDepField(entry *PackageEntry, kind graph.DepKind, deps map[string]string) {
	if len(deps) == 0 {
		return
	}
	switch kind {
	case graph.DepDev:
		entry.DevDependencies = cloneStringMap(deps)
	case graph.DepOptional:
		entry.OptionalDependencies = cloneStringMap(deps)
	case graph.DepPeer:
		entry.PeerDependencies = cloneStringMap(deps)
	default:
		entry.Dependencies = cloneStringMap(deps)
	}
}

func sortedKeys[T any](m map[string]T) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedStringKeys(m map[string]string) []string {
	return sortedKeys(m)
}
