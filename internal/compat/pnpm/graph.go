package pnpm

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/lockfile"
)

// ToGraph converts a pnpm document to the canonical graph.
func ToGraph(doc *Document) (*graph.Graph, error) {
	if doc == nil {
		return nil, apperr.New(apperr.Lockfile, "pnpm.graph", "document", "nil document")
	}
	if IsV6Layout(doc) {
		return v6ToGraph(doc)
	}
	return v9ShapeToGraph(doc)
}

func v6ToGraph(doc *Document) (*graph.Graph, error) {
	g := &graph.Graph{SchemaVersion: graph.SchemaVersion}
	g.Importers = []graph.Importer{{ID: graph.RootImporter, Path: "."}}

	pkgs := make([]graph.Package, 0, len(doc.Packages))
	pkgKeys := sortedStrings(keys(doc.Packages))
	for _, pathKey := range pkgKeys {
		entry := doc.Packages[pathKey]
		name, version, err := v6PathToNameVersion(pathKey)
		if err != nil {
			return nil, err
		}
		pkgs = append(pkgs, graph.Package{
			ID:        graph.PackageID{Name: name, Version: version},
			Integrity: integrityFromResolution(entry.Resolution),
		})
	}
	g.Packages = pkgs

	for name, dep := range doc.Dependencies {
		kind := graph.DepProd
		g.Edges = append(g.Edges, graph.Edge{
			From:  string(graph.RootImporter),
			Name:  name,
			To:    v6DepVersionToKey(name, dep.Version),
			Kind:  kind,
			Range: dep.Specifier,
		})
	}
	for _, pathKey := range pkgKeys {
		entry := doc.Packages[pathKey]
		fromKey, err := v6PathToKey(pathKey)
		if err != nil {
			return nil, err
		}
		for depName, depVer := range entry.Dependencies {
			g.Edges = append(g.Edges, graph.Edge{
				From: fromKey,
				Name: depName,
				To:   depVer,
				Kind: graph.DepProd,
			})
		}
	}
	if err := g.Validate(); err != nil {
		return nil, err
	}
	return g, nil
}

func v9ShapeToGraph(doc *Document) (*graph.Graph, error) {
	g := &graph.Graph{SchemaVersion: graph.SchemaVersion}
	importerIDs := sortedStrings(mapImporterSectionKeys(doc.Importers))
	if len(importerIDs) == 0 {
		importerIDs = []string{"."}
		doc.Importers["."] = ImporterSection{Dependencies: doc.Dependencies}
	}
	for _, id := range importerIDs {
		im := doc.Importers[id]
		g.Importers = append(g.Importers, graph.Importer{ID: graph.ImporterID(id), Path: id})
		appendImporterEdges(g, graph.ImporterID(id), im.Dependencies, graph.DepProd)
		appendImporterEdges(g, graph.ImporterID(id), im.DevDependencies, graph.DepDev)
	}

	pkgKeys := sortedStrings(keys(doc.Packages))
	for _, key := range pkgKeys {
		entry := doc.Packages[key]
		name, version := keyToNameVersion(key)
		g.Packages = append(g.Packages, graph.Package{
			ID:        graph.PackageID{Name: name, Version: version},
			Integrity: integrityFromResolution(entry.Resolution),
		})
		for depName, depVer := range entry.Dependencies {
			g.Edges = append(g.Edges, graph.Edge{
				From: key,
				Name: depName,
				To:   depVer,
				Kind: graph.DepProd,
			})
		}
	}
	if err := g.Validate(); err != nil {
		return nil, err
	}
	return g, nil
}

func appendImporterEdges(g *graph.Graph, from graph.ImporterID, deps map[string]ImporterDep, kind graph.DepKind) {
	for _, name := range sortedStrings(mapStringKeys(deps)) {
		dep := deps[name]
		g.Edges = append(g.Edges, graph.Edge{
			From:  string(from),
			Name:  name,
			To:    depVersionToKey(name, dep.Version),
			Kind:  kind,
			Range: dep.Specifier,
		})
	}
}

func depVersionToKey(name, version string) string {
	if strings.Contains(version, "@") {
		return version
	}
	return name + "@" + version
}

func v6DepVersionToKey(name, version string) string {
	if strings.Contains(version, "/") {
		parts := strings.Split(version, "/")
		if len(parts) >= 2 {
			return name + "@" + parts[len(parts)-1]
		}
	}
	return depVersionToKey(name, version)
}

func keyToDepVersion(name, key string) string {
	prefix := name + "@"
	if strings.HasPrefix(key, prefix) {
		return strings.TrimPrefix(key, prefix)
	}
	return key
}

// FromGraph builds a pnpm document from a validated graph and prior template.
func FromGraph(g *graph.Graph, prior *Document, det lockfile.Detection) (*Document, error) {
	if g == nil {
		return nil, apperr.New(apperr.Lockfile, "pnpm.graph", "graph", "nil graph")
	}
	if err := g.Validate(); err != nil {
		return nil, err
	}
	doc := &Document{
		Detection:  det,
		Settings:   map[string]any{},
		Importers:  map[string]ImporterSection{},
		Packages:   map[string]PackageEntry{},
		Snapshots:  map[string]map[string]any{},
		Extensions: lockfile.Extensions{},
	}
	if prior != nil {
		cloned := prior.Clone()
		doc.Settings = cloned.Settings
		doc.Extensions = cloned.Extensions
		doc.LockfileVersion = cloned.LockfileVersion
	}
	if doc.LockfileVersion == "" {
		doc.LockfileVersion = defaultLockfileVersion(det)
	}
	if det.Format == FormatV6 {
		return fromGraphV6(g, doc)
	}
	return fromGraphV9Shape(g, doc, prior)
}

func fromGraphV6(g *graph.Graph, doc *Document) (*Document, error) {
	doc.Dependencies = map[string]ImporterDep{}
	for _, e := range g.Edges {
		if graph.ImporterID(e.From) != graph.RootImporter {
			continue
		}
		doc.Dependencies[e.Name] = ImporterDep{Specifier: e.Range, Version: e.To}
	}
	for _, p := range g.Packages {
		pathKey := v6KeyToPath(p.ID.Key())
		entry := PackageEntry{
			Resolution: map[string]any{},
			Engines:    map[string]any{"node": ">=0.10.0"},
			Extra:      map[string]any{},
		}
		if p.Integrity != "" {
			entry.Resolution["integrity"] = p.Integrity
		}
		doc.Packages[pathKey] = entry
	}
	for _, e := range g.Edges {
		if graph.ImporterID(e.From) == graph.RootImporter {
			continue
		}
		pathKey := v6KeyToPath(e.From)
		entry := doc.Packages[pathKey]
		if entry.Dependencies == nil {
			entry.Dependencies = map[string]string{}
		}
		entry.Dependencies[e.Name] = e.To
		doc.Packages[pathKey] = entry
	}
	return doc, nil
}

func fromGraphV9Shape(g *graph.Graph, doc *Document, prior *Document) (*Document, error) {
	importers := map[string]ImporterSection{}
	for _, im := range g.Importers {
		importers[string(im.ID)] = ImporterSection{
			Dependencies:    map[string]ImporterDep{},
			DevDependencies: map[string]ImporterDep{},
		}
	}
	for _, e := range g.Edges {
		id := graph.ImporterID(e.From)
		sec, ok := importers[string(id)]
		if !ok {
			continue
		}
		dep := ImporterDep{Specifier: e.Range, Version: keyToDepVersion(e.Name, e.To)}
		switch e.Kind {
		case graph.DepDev:
			sec.DevDependencies[e.Name] = dep
		default:
			sec.Dependencies[e.Name] = dep
		}
		importers[string(id)] = sec
	}
	doc.Importers = importers

	priorPkgs := map[string]PackageEntry{}
	if prior != nil {
		priorPkgs = prior.Packages
	}
	for _, p := range g.Packages {
		key := p.ID.Key()
		entry := PackageEntry{
			Resolution: map[string]any{},
			Engines:    map[string]any{},
			Extra:      map[string]any{},
		}
		if prev, ok := priorPkgs[key]; ok {
			entry = prev
			if entry.Resolution == nil {
				entry.Resolution = map[string]any{}
			}
		}
		if p.Integrity != "" {
			entry.Resolution["integrity"] = p.Integrity
		}
		doc.Packages[key] = entry
		if _, ok := doc.Snapshots[key]; !ok {
			doc.Snapshots[key] = map[string]any{}
		}
	}
	return doc, nil
}

func defaultLockfileVersion(det lockfile.Detection) string {
	switch det.Format {
	case FormatV6:
		return "5.4"
	case FormatV10, FormatV11:
		return "9.0"
	default:
		return "9.0"
	}
}

func integrityFromResolution(res map[string]any) string {
	if res == nil {
		return ""
	}
	if v, ok := res["integrity"].(string); ok {
		return v
	}
	return ""
}

func v6PathToNameVersion(pathKey string) (string, string, error) {
	key, err := v6PathToKey(pathKey)
	if err != nil {
		return "", "", err
	}
	n, v := keyToNameVersion(key)
	return n, v, nil
}

func v6PathToKey(pathKey string) (string, error) {
	pathKey = strings.TrimPrefix(pathKey, "/")
	parts := strings.Split(pathKey, "/")
	if len(parts) < 2 {
		return "", apperr.New(apperr.Lockfile, "pnpm.graph", pathKey, "invalid v6 package path")
	}
	version := parts[len(parts)-1]
	name := strings.Join(parts[:len(parts)-1], "/")
	return name + "@" + version, nil
}

func v6KeyToPath(key string) string {
	name, version := keyToNameVersion(key)
	if strings.HasPrefix(name, "@") {
		return "/" + name + "/" + version
	}
	return "/" + name + "/" + version
}

func keyToNameVersion(key string) (string, string) {
	if strings.HasPrefix(key, "@") {
		slash := strings.IndexByte(key, '/')
		if slash < 0 {
			return key, ""
		}
		at := strings.LastIndexByte(key[slash:], '@')
		if at < 0 {
			return key, ""
		}
		return key[:slash+at], key[slash+at+1:]
	}
	at := strings.LastIndexByte(key, '@')
	if at < 0 {
		return key, ""
	}
	return key[:at], key[at+1:]
}

// GraphsEqual compares two graphs for lock write decisions, ignoring fetch-only metadata.
func GraphsEqual(a, b *graph.Graph) (bool, error) {
	ac, err := cloneGraphStripFetchMeta(a)
	if err != nil {
		return false, err
	}
	bc, err := cloneGraphStripFetchMeta(b)
	if err != nil {
		return false, err
	}
	ga, err := graph.EncodeJSON(ac)
	if err != nil {
		return false, err
	}
	gb, err := graph.EncodeJSON(bc)
	if err != nil {
		return false, err
	}
	return string(ga) == string(gb), nil
}

func cloneGraphStripFetchMeta(g *graph.Graph) (*graph.Graph, error) {
	if g == nil {
		return nil, nil
	}
	data, err := graph.EncodeJSON(g)
	if err != nil {
		return nil, err
	}
	out, err := graph.DecodeJSON(data)
	if err != nil {
		return nil, err
	}
	for i := range out.Packages {
		out.Packages[i].TarballURL = ""
	}
	for i := range out.Importers {
		out.Importers[i].Name = ""
	}
	return out, nil
}

func sortedStrings(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

func mapImporterSectionKeys(m map[string]ImporterSection) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func mapImporterKeys(m map[string]ImporterSection) []string {
	return mapImporterSectionKeys(m)
}

func mapPackageKeys(m map[string]PackageEntry) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func keys(m map[string]PackageEntry) []string {
	return mapPackageKeys(m)
}

func mapStringKeys(m map[string]ImporterDep) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func mapAnyKeys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func jsonUnmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
