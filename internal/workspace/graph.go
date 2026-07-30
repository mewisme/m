package workspace

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/manifest"
)

// Member is one workspace package discovered under the project root.
type Member struct {
	Name    string
	Version string
	Path    string // root-relative POSIX path
}

// WorkspaceGraph wraps a workspace Index with member metadata and dependency order.
type WorkspaceGraph struct {
	Root    string
	Index   *Index
	Members map[string]Member // package name -> member
	ByPath  map[string]Member // member path -> member
}

// BuildGraph loads workspace members and rejects duplicate package names.
func BuildGraph(root string) (*WorkspaceGraph, error) {
	idx, err := BuildIndex(root)
	if err != nil {
		return nil, err
	}
	g := &WorkspaceGraph{
		Root:    idx.Root,
		Index:   idx,
		Members: make(map[string]Member),
		ByPath:  make(map[string]Member),
	}
	for _, memPath := range idx.Members {
		doc, err := manifest.Load(filepath.Join(root, filepath.FromSlash(memPath), "package.json"))
		if err != nil {
			return nil, apperr.Wrap(apperr.Manifest, "workspace.graph", memPath, err)
		}
		name := doc.Name
		if name == "" {
			return nil, apperr.New(apperr.Manifest, "workspace.graph", memPath, "workspace member missing name")
		}
		if prev, ok := g.Members[name]; ok {
			return nil, apperr.New(apperr.Resolve, "workspace.graph", name,
				fmt.Sprintf("ambiguous workspace target %q: members %q and %q", name, prev.Path, memPath))
		}
		ver := doc.Version
		if ver == "" {
			ver = "0.0.0"
		}
		m := Member{Name: name, Version: ver, Path: memPath}
		g.Members[name] = m
		g.ByPath[memPath] = m
	}
	return g, nil
}

// MemberPaths returns sorted member paths (including ".").
func (g *WorkspaceGraph) MemberPaths() []string {
	if g == nil || g.Index == nil {
		return nil
	}
	return append([]string(nil), g.Index.Members...)
}

// TopoOrder returns member paths in topological order (dependencies before dependents).
func (g *WorkspaceGraph) TopoOrder() ([]string, error) {
	if g == nil || len(g.ByPath) == 0 {
		return nil, nil
	}
	edges := map[string][]string{}
	inDegree := map[string]int{}
	for path := range g.ByPath {
		inDegree[path] = 0
	}
	for path := range g.ByPath {
		doc, err := manifest.Load(filepath.Join(g.Root, filepath.FromSlash(path), "package.json"))
		if err != nil {
			return nil, apperr.Wrap(apperr.Manifest, "workspace.topo", path, err)
		}
		for _, depMap := range []map[string]string{doc.Dependencies, doc.DevDependencies, doc.OptionalDependencies} {
			for depName, spec := range depMap {
				target, ok := WorkspaceTargetForDep(depName, spec, g)
				if !ok {
					continue
				}
				if target.Path == path {
					continue
				}
				edges[target.Path] = append(edges[target.Path], path)
				inDegree[path]++
			}
		}
	}
	var queue []string
	for path, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, path)
		}
	}
	sort.Strings(queue)
	var out []string
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		out = append(out, cur)
		for _, next := range edges[cur] {
			inDegree[next]--
			if inDegree[next] == 0 {
				queue = append(queue, next)
			}
		}
		sort.Strings(queue)
	}
	if len(out) != len(g.ByPath) {
		return nil, apperr.New(apperr.Resolve, "workspace.topo", g.Root, "cyclic workspace dependency graph")
	}
	return out, nil
}

// SelectedWorkspaceDeps returns sorted workspace dependency paths of path that are in selected.
func SelectedWorkspaceDeps(g *WorkspaceGraph, path string, selected map[string]struct{}) []string {
	if g == nil || selected == nil {
		return nil
	}
	var out []string
	for _, dep := range workspaceDepsOf(g, path) {
		if _, ok := selected[dep]; ok {
			out = append(out, dep)
		}
	}
	return out
}

// InducedSubgraph returns dependency edges and in-degrees for selected members only.
// edges maps dependency path -> dependent paths (dependency completes before dependent).
func InducedSubgraph(g *WorkspaceGraph, selected []string) (edges map[string][]string, inDegree map[string]int, err error) {
	if g == nil {
		return nil, nil, apperr.New(apperr.Internal, "workspace.graph", "", "nil graph")
	}
	sel := make(map[string]struct{}, len(selected))
	for _, p := range selected {
		if p == "." {
			continue
		}
		if _, ok := g.ByPath[p]; !ok {
			return nil, nil, apperr.New(apperr.NotFound, "workspace.induced", p, "workspace member not found")
		}
		sel[p] = struct{}{}
	}
	edges = map[string][]string{}
	inDegree = map[string]int{}
	for p := range sel {
		inDegree[p] = 0
	}
	for p := range sel {
		for _, dep := range SelectedWorkspaceDeps(g, p, sel) {
			edges[dep] = append(edges[dep], p)
			inDegree[p]++
		}
	}
	for dep := range edges {
		sort.Strings(edges[dep])
	}
	return edges, inDegree, nil
}

// ValidateSelectedCycle rejects cycles inside the selected induced subgraph.
func ValidateSelectedCycle(g *WorkspaceGraph, selected []string) error {
	if g == nil || len(selected) == 0 {
		return nil
	}
	edges, inDegree, err := InducedSubgraph(g, selected)
	if err != nil {
		return err
	}
	queue := make([]string, 0, len(inDegree))
	for p, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, p)
		}
	}
	sort.Strings(queue)
	var visited int
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		visited++
		for _, next := range edges[cur] {
			inDegree[next]--
			if inDegree[next] == 0 {
				queue = append(queue, next)
			}
		}
		sort.Strings(queue)
	}
	if visited != len(inDegree) {
		return apperr.New(apperr.Resolve, "workspace.graph", g.Root, "cyclic workspace dependency graph in selected packages")
	}
	return nil
}

// ReverseTopoOrder returns member paths in reverse topological order for the given subset.
func (g *WorkspaceGraph) ReverseTopoOrder(paths []string) ([]string, error) {
	order, err := g.TopoOrderFor(paths)
	if err != nil {
		return nil, err
	}
	for i, j := 0, len(order)-1; i < j; i, j = i+1, j-1 {
		order[i], order[j] = order[j], order[i]
	}
	return order, nil
}

// TopoOrderFor returns topological order restricted to paths (dependencies before dependents).
func (g *WorkspaceGraph) TopoOrderFor(paths []string) ([]string, error) {
	if g == nil || len(paths) == 0 {
		return nil, nil
	}
	edges, inDegree, err := InducedSubgraph(g, paths)
	if err != nil {
		return nil, err
	}
	queue := make([]string, 0, len(inDegree))
	for p, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, p)
		}
	}
	sort.Strings(queue)
	var out []string
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		out = append(out, cur)
		for _, next := range edges[cur] {
			inDegree[next]--
			if inDegree[next] == 0 {
				queue = append(queue, next)
			}
		}
		sort.Strings(queue)
	}
	if len(out) != len(inDegree) {
		return nil, apperr.New(apperr.Resolve, "workspace.topo", g.Root, "cyclic workspace dependency graph")
	}
	return out, nil
}

// WorkspaceTargetForDep returns the workspace member targeted by a workspace protocol dep on name.
func WorkspaceTargetForDep(depName, spec string, g *WorkspaceGraph) (Member, bool) {
	spec = strings.TrimSpace(spec)
	if !strings.HasPrefix(spec, "workspace:") {
		return Member{}, false
	}
	rng := spec[len("workspace:"):]
	if rng != "*" && rng != "^" {
		return Member{}, false
	}
	if g == nil {
		return Member{}, false
	}
	mem, ok := g.Members[depName]
	return mem, ok
}
