package pnpm

import (
	"encoding/json"
	"fmt"
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
	if err := ValidateSupportedPnpm(doc); err != nil {
		return nil, err
	}
	return v9ShapeToGraph(doc)
}

func v9ShapeToGraph(doc *Document) (*graph.Graph, error) {
	g := &graph.Graph{SchemaVersion: graph.SchemaVersion}

	instanceKeys := buildInstanceSet(doc)
	packageKeys := make(map[string]struct{}, len(instanceKeys))
	graphKeys := make([]string, 0, len(instanceKeys))
	for _, instanceKey := range instanceKeys {
		pkgID, err := packageIDFromInstanceKey(instanceKey)
		if err != nil {
			return nil, err
		}
		baseKey := basePackageKeyFromInstance(instanceKey)
		entry, hasBase := doc.Packages[baseKey]
		if !hasBase && !isProtocolRef(instanceKey) {
			return nil, apperr.New(apperr.Lockfile, "pnpm.graph", instanceKey,
				"missing base package metadata for "+baseKey)
		}
		graphKey := pkgID.Key()
		g.Packages = append(g.Packages, graph.Package{
			ID:        pkgID,
			Integrity: integrityFromResolution(entry.Resolution),
		})
		packageKeys[graphKey] = struct{}{}
		graphKeys = append(graphKeys, graphKey)
	}
	idx := NewPackageIndex(appendWorkspaceIndexKeys(graphKeys, doc))

	importerIDs := sortedStrings(mapImporterSectionKeys(doc.Importers))
	if len(importerIDs) == 0 {
		importerIDs = []string{"."}
		doc.Importers["."] = ImporterSection{Dependencies: doc.Dependencies}
	}
	for _, id := range importerIDs {
		im := doc.Importers[id]
		g.Importers = append(g.Importers, graph.Importer{ID: graph.ImporterID(id), Path: id})
		if err := appendImporterEdges(g, graph.ImporterID(id), im.Dependencies, graph.DepProd, idx); err != nil {
			return nil, err
		}
		if err := appendImporterEdges(g, graph.ImporterID(id), im.DevDependencies, graph.DepDev, idx); err != nil {
			return nil, err
		}
		if err := appendImporterEdges(g, graph.ImporterID(id), im.OptionalDependencies, graph.DepOptional, idx); err != nil {
			return nil, err
		}
	}

	if len(doc.Snapshots) > 0 {
		for _, snapKey := range sortedSnapshotKeys(doc.Snapshots) {
			graphFrom, err := instanceKeyToGraphKey(snapKey)
			if err != nil {
				return nil, err
			}
			if err := appendSnapshotEdges(g, graphFrom, doc.Snapshots[snapKey], packageKeys, idx); err != nil {
				return nil, err
			}
		}
	} else {
		for _, instanceKey := range instanceKeys {
			baseKey := basePackageKeyFromInstance(instanceKey)
			entry, ok := doc.Packages[baseKey]
			if !ok {
				continue
			}
			graphFrom, err := instanceKeyToGraphKey(instanceKey)
			if err != nil {
				return nil, err
			}
			for depName, depVer := range entry.Dependencies {
				g.Edges = append(g.Edges, graph.Edge{
					From: graphFrom,
					Name: depName,
					To:   depVer,
					Kind: graph.DepProd,
				})
			}
		}
	}

	if err := validateImporterTargets(g, doc); err != nil {
		return nil, err
	}
	if err := g.Validate(); err != nil {
		return nil, err
	}
	return g, nil
}

func appendSnapshotEdges(g *graph.Graph, from string, snap map[string]any, known map[string]struct{}, idx PackageIndex) error {
	if err := appendSnapshotDepField(g, from, snap, "dependencies", graph.DepProd, false, known, idx); err != nil {
		return err
	}
	if err := appendSnapshotDepField(g, from, snap, "optionalDependencies", graph.DepOptional, true, known, idx); err != nil {
		return err
	}
	if err := appendSnapshotDepField(g, from, snap, "peerDependencies", graph.DepPeer, false, known, idx); err != nil {
		return err
	}
	return nil
}

func appendSnapshotDepField(g *graph.Graph, from string, snap map[string]any, field string, kind graph.DepKind, optional bool, known map[string]struct{}, idx PackageIndex) error {
	raw, ok := snap[field]
	if !ok {
		return nil
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return apperr.New(apperr.Lockfile, "pnpm.graph", from+"."+field, "expected mapping")
	}
	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		to, ok := m[name].(string)
		if !ok || to == "" {
			return apperr.New(apperr.Lockfile, "pnpm.graph", from+"."+field+"."+name, "expected version key")
		}
		target, err := ResolveDependencyTarget(name, to, idx)
		if err != nil {
			return err
		}
		if !isAllowedEdgeTarget(target.Key, known) {
			return apperr.New(apperr.Lockfile, "pnpm.graph", from, "snapshot edge target not found: "+target.Key)
		}
		g.Edges = append(g.Edges, graph.Edge{
			From:     from,
			Name:     name,
			To:       target.Key,
			Kind:     kind,
			Optional: optional,
		})
	}
	return nil
}

func isAllowedEdgeTarget(to string, known map[string]struct{}) bool {
	if _, ok := known[to]; ok {
		return true
	}
	return strings.HasPrefix(to, "link:") || strings.HasPrefix(to, "workspace:")
}

func validateImporterTargets(g *graph.Graph, doc *Document) error {
	packageKeys := make(map[string]struct{}, len(g.Packages))
	for _, p := range g.Packages {
		packageKeys[p.ID.Key()] = struct{}{}
	}
	for _, e := range g.Edges {
		if _, ok := packageKeys[e.From]; ok {
			continue
		}
		if _, ok := doc.Importers[e.From]; ok {
			if !isAllowedEdgeTarget(e.To, packageKeys) {
				return apperr.New(apperr.Lockfile, "pnpm.graph", e.From, "importer edge target not found: "+e.To)
			}
		}
	}
	return nil
}

func sortedSnapshotKeys(m map[string]map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func appendImporterEdges(g *graph.Graph, from graph.ImporterID, deps map[string]ImporterDep, kind graph.DepKind, idx PackageIndex) error {
	for _, name := range sortedStrings(mapStringKeys(deps)) {
		dep := deps[name]
		target, err := ResolveDependencyTarget(name, dep.Version, idx)
		if err != nil {
			return apperr.Wrap(apperr.Lockfile, "pnpm.graph", string(from)+"."+name, err)
		}
		g.Edges = append(g.Edges, graph.Edge{
			From:  string(from),
			Name:  name,
			To:    target.Key,
			Kind:  kind,
			Range: dep.Specifier,
		})
	}
	return nil
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
	return fromGraphV9Shape(g, doc, prior)
}

func fromGraphV9Shape(g *graph.Graph, doc *Document, prior *Document) (*Document, error) {
	importers := map[string]ImporterSection{}
	for _, im := range g.Importers {
		id := string(im.ID)
		sec := ImporterSection{
			Dependencies:         map[string]ImporterDep{},
			DevDependencies:      map[string]ImporterDep{},
			OptionalDependencies: map[string]ImporterDep{},
		}
		if prior != nil {
			if prev, ok := prior.Importers[id]; ok {
				sec.DependenciesMeta = cloneMap(prev.DependenciesMeta)
				sec.PublishDirectory = prev.PublishDirectory
				if prev.Extra != nil {
					sec.Extra = make(map[string]json.RawMessage, len(prev.Extra))
					for k, v := range prev.Extra {
						sec.Extra[k] = v
					}
				}
			}
		}
		importers[id] = sec
	}
	for _, e := range g.Edges {
		id := graph.ImporterID(e.From)
		sec, ok := importers[string(id)]
		if !ok {
			continue
		}
		dep := ImporterDep{Specifier: e.Range}
		var err error
		dep.Version, err = EncodeDependencyRef(e.Name, e.To)
		if err != nil {
			return nil, apperr.Wrap(apperr.Lockfile, "pnpm.graph", string(id)+"."+e.Name, err)
		}
		switch e.Kind {
		case graph.DepDev:
			sec.DevDependencies[e.Name] = dep
		case graph.DepOptional:
			sec.OptionalDependencies[e.Name] = dep
		default:
			sec.Dependencies[e.Name] = dep
		}
		importers[string(id)] = sec
	}
	doc.Importers = importers

	priorPkgs := map[string]PackageEntry{}
	priorSnaps := map[string]map[string]any{}
	if prior != nil {
		priorPkgs = prior.Packages
		priorSnaps = prior.Snapshots
	}
	for _, p := range g.Packages {
		graphKey := p.ID.Key()
		if isProtocolRef(graphKey) {
			continue
		}
		instanceKey, err := graphKeyToInstanceKey(graphKey)
		if err != nil {
			return nil, err
		}
		baseKey := basePackageKeyFromInstance(instanceKey)
		entry := PackageEntry{
			Resolution: map[string]any{},
			Engines:    map[string]any{},
			Extra:      map[string]any{},
		}
		if prev, ok := priorPkgs[baseKey]; ok {
			entry = prev
			if entry.Resolution == nil {
				entry.Resolution = map[string]any{}
			}
		}
		if p.Integrity != "" {
			entry.Resolution["integrity"] = p.Integrity
		}
		doc.Packages[baseKey] = entry
		priorSnap := priorSnaps[instanceKey]
		if priorSnap == nil {
			priorSnap = priorSnaps[baseKey]
		}
		snap, err := snapshotFromGraphEdges(graphKey, g, priorSnap)
		if err != nil {
			return nil, err
		}
		doc.Snapshots[instanceKey] = snap
	}
	return doc, nil
}

func snapshotFromGraphEdges(pkgKey string, g *graph.Graph, prior map[string]any) (map[string]any, error) {
	snap := map[string]any{}
	for k, v := range prior {
		if k != "dependencies" && k != "optionalDependencies" && k != "peerDependencies" {
			snap[k] = v
		}
	}
	deps := map[string]string{}
	opt := map[string]string{}
	peer := map[string]string{}
	for _, e := range g.Edges {
		if e.From != pkgKey {
			continue
		}
		ref, err := EncodeDependencyRef(e.Name, e.To)
		if err != nil {
			return nil, apperr.Wrap(apperr.Lockfile, "pnpm.graph", pkgKey+"."+e.Name,
				fmt.Errorf("encode edge to %q (%s): %w", e.To, e.Kind, err))
		}
		switch e.Kind {
		case graph.DepOptional:
			opt[e.Name] = ref
		case graph.DepPeer:
			peer[e.Name] = ref
		default:
			deps[e.Name] = ref
		}
	}
	if len(deps) > 0 {
		snap["dependencies"] = deps
	}
	if len(opt) > 0 {
		snap["optionalDependencies"] = opt
	}
	if len(peer) > 0 {
		snap["peerDependencies"] = peer
	}
	return snap, nil
}

func appendWorkspaceIndexKeys(graphKeys []string, doc *Document) []string {
	seen := make(map[string]struct{}, len(graphKeys))
	for _, k := range graphKeys {
		seen[k] = struct{}{}
	}
	for _, im := range doc.Importers {
		collectLinkRefs(im.Dependencies, seen)
		collectLinkRefs(im.DevDependencies, seen)
		collectLinkRefs(im.OptionalDependencies, seen)
	}
	if len(seen) == len(graphKeys) {
		return graphKeys
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	return sortedStrings(out)
}

func collectLinkRefs(deps map[string]ImporterDep, seen map[string]struct{}) {
	for _, dep := range deps {
		if isLocalProtocolRef(dep.Version) {
			seen[dep.Version] = struct{}{}
		}
	}
}

func defaultLockfileVersion(det lockfile.Detection) string {
	return SelectPolicy(det).DefaultLockfileVersion()
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
