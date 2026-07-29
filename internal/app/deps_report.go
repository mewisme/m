package app

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/project"
	"github.com/mewisme/mew/internal/registry"
	"github.com/mewisme/mew/internal/resolver"
	"github.com/mewisme/mew/internal/semver"
	"github.com/mewisme/mew/internal/workspace"
)

// DepTreeOptions controls dependency tree construction.
type DepTreeOptions struct {
	ImporterID graph.ImporterID
	ProdOnly   bool
	Depth      int // dependency levels below root; -1 unlimited
}

// DepTreeNode is one node in a dependency tree (JSON output).
type DepTreeNode struct {
	Name         string        `json:"name"`
	Version      string        `json:"version,omitempty"`
	Dependencies []DepTreeNode `json:"dependencies,omitempty"`
}

// LoadInstalledGraph reads the incumbent lock graph for a project.
func LoadInstalledGraph(ctx context.Context, ac *Context, proj *project.Project) (*graph.Graph, error) {
	g, err := readLockHints(ctx, ac, proj)
	if err != nil {
		return nil, err
	}
	if g == nil {
		path := LockPath(proj)
		return nil, apperr.New(apperr.NotFound, "app.deps", path, "lockfile not found")
	}
	return g, nil
}

// BuildDepTree builds a dependency tree for one importer from a lock graph.
func BuildDepTree(g *graph.Graph, opts DepTreeOptions) (DepTreeNode, error) {
	if g == nil {
		return DepTreeNode{}, apperr.New(apperr.Lockfile, "app.deps.tree", "graph", "nil graph")
	}
	importer := opts.ImporterID
	if importer == "" {
		importer = graph.RootImporter
	}
	found := false
	for _, im := range g.Importers {
		if im.ID == importer {
			found = true
			break
		}
	}
	if !found {
		return DepTreeNode{}, apperr.New(apperr.NotFound, "app.deps.tree", string(importer), "importer not in lock graph")
	}
	pkgByKey := indexPackages(g)
	childEdges := indexChildEdges(g)
	seen := make(map[string]struct{})
	root := buildDepTreeNode(string(importer), pkgByKey, childEdges, opts, 0, seen)
	return root, nil
}

func buildDepTreeNode(
	from string,
	pkgByKey map[string]graph.Package,
	childEdges map[string][]graph.Edge,
	opts DepTreeOptions,
	depth int,
	seen map[string]struct{},
) DepTreeNode {
	node := DepTreeNode{}
	if pkg, ok := pkgByKey[from]; ok {
		node.Name = pkg.ID.Name
		node.Version = pkg.ID.Version
	}
	if opts.Depth >= 0 && depth >= opts.Depth {
		return node
	}
	edges := childEdges[from]
	sort.SliceStable(edges, func(i, j int) bool {
		a, b := edges[i], edges[j]
		if a.Name != b.Name {
			return a.Name < b.Name
		}
		return a.To < b.To
	})
	for _, e := range edges {
		if opts.ProdOnly && e.Kind != graph.DepProd {
			continue
		}
		if _, ok := seen[e.To]; ok {
			continue
		}
		seen[e.To] = struct{}{}
		child := buildDepTreeNode(e.To, pkgByKey, childEdges, opts, depth+1, seen)
		if child.Name == "" {
			child.Name = graph.TargetNameFromKey(e.To)
			if pkg, ok := pkgByKey[e.To]; ok {
				child.Version = pkg.ID.Version
			}
		}
		node.Dependencies = append(node.Dependencies, child)
	}
	return node
}

// FormatDepTreeText renders an npm-like dependency tree.
func FormatDepTreeText(rootName, rootVersion string, tree DepTreeNode) string {
	var b strings.Builder
	b.WriteString(rootName)
	if rootVersion != "" {
		b.WriteString("@")
		b.WriteString(rootVersion)
	}
	b.WriteByte('\n')
	writeDepTreeLines(&b, tree.Dependencies, "", true)
	return strings.TrimRight(b.String(), "\n")
}

func writeDepTreeLines(b *strings.Builder, nodes []DepTreeNode, prefix string, top bool) {
	for i, n := range nodes {
		last := i == len(nodes)-1
		branch := "└── "
		nextPrefix := prefix + "    "
		if !last {
			branch = "├── "
			nextPrefix = prefix + "│   "
		}
		if top && len(nodes) == 1 {
			branch = "└── "
			nextPrefix = prefix + "    "
		}
		b.WriteString(prefix)
		b.WriteString(branch)
		b.WriteString(n.Name)
		if n.Version != "" {
			b.WriteString("@")
			b.WriteString(n.Version)
		}
		b.WriteByte('\n')
		if len(n.Dependencies) > 0 {
			writeDepTreeLines(b, n.Dependencies, nextPrefix, false)
		}
	}
}

// OutdatedOptions controls m outdated.
type OutdatedOptions struct {
	Recursive bool
	Filter    []string
}

// OutdatedEntry is one outdated package row.
type OutdatedEntry struct {
	Package          string `json:"package"`
	Importer         string `json:"importer,omitempty"`
	Current          string `json:"current"`
	Wanted           string `json:"wanted"`
	Latest           string `json:"latest"`
	DependencyType   string `json:"dependencyType,omitempty"`
	DependentPackage string `json:"dependentPackage,omitempty"`
}

// OutdatedReport is the full outdated result.
type OutdatedReport struct {
	Entries []OutdatedEntry `json:"entries"`
}

// Outdated compares locked versions against manifest ranges and registry metadata.
func Outdated(ctx context.Context, ac *Context, proj *project.Project, opts OutdatedOptions) (OutdatedReport, error) {
	if ac == nil || proj == nil {
		return OutdatedReport{}, apperr.New(apperr.Internal, "app.outdated", "", "missing context or project")
	}
	g, err := LoadInstalledGraph(ctx, ac, proj)
	if err != nil {
		return OutdatedReport{}, err
	}
	importers, err := resolveOutdatedImporters(proj, ac, opts)
	if err != nil {
		return OutdatedReport{}, err
	}
	eng, err := resolver.NewFromApp(ac.Config, proj)
	if err != nil {
		return OutdatedReport{}, err
	}
	pkgByKey := indexPackages(g)
	childEdges := indexChildEdges(g)
	var entries []OutdatedEntry
	for _, im := range importers {
		refs := collectOutdatedRefs(g, im.ID, opts.Recursive, pkgByKey, childEdges)
		for _, ref := range refs {
			entry, ok, err := outdatedEntryForRef(ctx, ac, proj, eng, ref)
			if err != nil {
				return OutdatedReport{}, err
			}
			if ok {
				entries = append(entries, entry)
			}
		}
	}
	sort.SliceStable(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if a.Importer != b.Importer {
			return a.Importer < b.Importer
		}
		return a.Package < b.Package
	})
	return OutdatedReport{Entries: entries}, nil
}

func resolveOutdatedImporters(proj *project.Project, ac *Context, opts OutdatedOptions) ([]graph.Importer, error) {
	g, err := readLockHints(ac.Ctx, ac, proj)
	if err != nil {
		return nil, err
	}
	if g == nil {
		return nil, apperr.New(apperr.NotFound, "app.outdated", LockPath(proj), "lockfile not found")
	}
	if len(opts.Filter) > 0 {
		if !workspace.Enabled(ac.Config) {
			return nil, apperr.New(apperr.Usage, "app.outdated", "--filter",
				"workspaces not enabled; set MEW_EXPERIMENTAL_WORKSPACES=1 or workspaces.enabled in config")
		}
		wg, err := workspace.BuildGraph(proj.Root)
		if err != nil {
			return nil, err
		}
		ids, err := workspace.ExpandFilter(wg, opts.Filter)
		if err != nil {
			return nil, err
		}
		return importersForIDs(g, ids)
	}
	if opts.Recursive && workspace.Enabled(ac.Config) {
		return append([]graph.Importer(nil), g.Importers...), nil
	}
	for _, im := range g.Importers {
		if im.ID == graph.RootImporter {
			return []graph.Importer{im}, nil
		}
	}
	if len(g.Importers) > 0 {
		return []graph.Importer{g.Importers[0]}, nil
	}
	return nil, apperr.New(apperr.NotFound, "app.outdated", "importer", "no importers in lock graph")
}

func importersForIDs(g *graph.Graph, ids []graph.ImporterID) ([]graph.Importer, error) {
	want := make(map[graph.ImporterID]struct{}, len(ids))
	for _, id := range ids {
		want[id] = struct{}{}
	}
	var out []graph.Importer
	for _, im := range g.Importers {
		if _, ok := want[im.ID]; ok {
			out = append(out, im)
		}
	}
	if len(out) == 0 {
		return nil, apperr.New(apperr.NotFound, "app.outdated", "importer", "filter matched no lock importers")
	}
	return out, nil
}

type outdatedRef struct {
	importer graph.ImporterID
	name     string
	current  string
	spec     string
	kind     graph.DepKind
	from     string
}

func collectOutdatedRefs(
	g *graph.Graph,
	importer graph.ImporterID,
	recursive bool,
	pkgByKey map[string]graph.Package,
	childEdges map[string][]graph.Edge,
) []outdatedRef {
	seen := make(map[string]outdatedRef)
	queue := []string{string(importer)}
	visited := map[string]struct{}{string(importer): {}}
	for len(queue) > 0 {
		from := queue[0]
		queue = queue[1:]
		edges := childEdges[from]
		sort.SliceStable(edges, func(i, j int) bool {
			return edges[i].Name < edges[j].Name
		})
		for _, e := range edges {
			pkg, ok := pkgByKey[e.To]
			if !ok {
				continue
			}
			key := string(importer) + "|" + pkg.ID.Name
			if _, exists := seen[key]; !exists {
				seen[key] = outdatedRef{
					importer: importer,
					name:     pkg.ID.Name,
					current:  pkg.ID.Version,
					spec:     e.Range,
					kind:     e.Kind,
					from:     e.From,
				}
			}
			if recursive {
				if _, ok := visited[e.To]; !ok {
					visited[e.To] = struct{}{}
					queue = append(queue, e.To)
				}
			}
		}
	}
	out := make([]outdatedRef, 0, len(seen))
	for _, ref := range seen {
		out = append(out, ref)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].name < out[j].name
	})
	return out
}

func outdatedEntryForRef(
	ctx context.Context,
	ac *Context,
	proj *project.Project,
	eng *resolver.Engine,
	ref outdatedRef,
) (OutdatedEntry, bool, error) {
	base := registry.ResolveBaseForPackage(ac.Config, proj.Root, proj.Identity, ref.name)
	pack, err := eng.Client.Packument(ctx, base, ref.name)
	if err != nil {
		return OutdatedEntry{}, false, err
	}
	wantedMeta, err := registry.SelectMaxSatisfying(pack, ref.spec)
	if err != nil {
		return OutdatedEntry{}, false, apperr.Wrap(apperr.Resolve, "app.outdated", ref.name, err)
	}
	latestMeta, err := pack.SelectVersion("latest")
	if err != nil {
		return OutdatedEntry{}, false, apperr.Wrap(apperr.Resolve, "app.outdated", ref.name, err)
	}
	wanted := wantedMeta.Version
	latest := latestMeta.Version
	if !isSemverLess(ref.current, wanted) && !isSemverLess(ref.current, latest) {
		return OutdatedEntry{}, false, nil
	}
	entry := OutdatedEntry{
		Package:  ref.name,
		Importer: string(ref.importer),
		Current:  ref.current,
		Wanted:   wanted,
		Latest:   latest,
	}
	if ref.kind != "" {
		entry.DependencyType = string(ref.kind)
	}
	if ref.from != string(ref.importer) {
		entry.DependentPackage = graph.TargetNameFromKey(ref.from)
	}
	return entry, true, nil
}

func isSemverLess(current, other string) bool {
	if current == other {
		return false
	}
	cmp, err := semver.Compare(current, other)
	if err != nil {
		return current < other
	}
	return cmp < 0
}

func indexPackages(g *graph.Graph) map[string]graph.Package {
	out := make(map[string]graph.Package, len(g.Packages))
	for _, p := range g.Packages {
		out[p.ID.Key()] = p
	}
	return out
}

func indexChildEdges(g *graph.Graph) map[string][]graph.Edge {
	out := make(map[string][]graph.Edge)
	for _, e := range g.Edges {
		out[e.From] = append(out[e.From], e)
	}
	return out
}

// ImporterLabel returns display name and version for an importer.
func ImporterLabel(proj *project.Project, id graph.ImporterID) (name, version string) {
	if proj != nil && proj.Doc != nil && id == graph.RootImporter {
		return proj.Doc.Name, proj.Doc.Version
	}
	if proj != nil && proj.Normalized != nil && id == graph.RootImporter {
		return proj.Normalized.Name, proj.Doc.Version
	}
	return string(id), ""
}

// FormatOutdatedTable renders a simple text table for m outdated.
func FormatOutdatedTable(entries []OutdatedEntry) string {
	if len(entries) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("Package  Current  Wanted  Latest  Location\n")
	for _, e := range entries {
		loc := e.Importer
		if loc == "" || loc == "." {
			loc = "."
		}
		fmt.Fprintf(&b, "%s  %s  %s  %s  %s\n", e.Package, e.Current, e.Wanted, e.Latest, loc)
	}
	return strings.TrimRight(b.String(), "\n")
}
