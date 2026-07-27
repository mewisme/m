package workspace

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/manifest"
)

// ExpandFilter resolves --filter patterns to sorted importer IDs.
func ExpandFilter(g *WorkspaceGraph, patterns []string) ([]graph.ImporterID, error) {
	if g == nil || len(g.ByPath) == 0 {
		return nil, apperr.New(apperr.Manifest, "workspace.filter", "", "not a workspace project")
	}
	if len(patterns) == 0 {
		return nil, apperr.New(apperr.Usage, "workspace.filter", "", "empty filter pattern")
	}
	var includes, excludes []string
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if strings.HasPrefix(p, "!") {
			excludes = append(excludes, p[1:])
		} else {
			includes = append(includes, p)
		}
	}
	if len(includes) == 0 {
		return nil, apperr.New(apperr.Usage, "workspace.filter", "", "filter requires at least one positive pattern")
	}
	matched := make(map[string]struct{})
	for _, pat := range includes {
		set, err := expandFilterPattern(g, pat)
		if err != nil {
			return nil, err
		}
		for p := range set {
			matched[p] = struct{}{}
		}
	}
	for _, pat := range excludes {
		set, err := expandFilterPattern(g, pat)
		if err != nil {
			return nil, err
		}
		for p := range set {
			delete(matched, p)
		}
	}
	if len(matched) == 0 {
		return nil, apperr.New(apperr.NotFound, "workspace.filter", strings.Join(patterns, ","), "no workspace packages matched filter")
	}
	out := make([]graph.ImporterID, 0, len(matched))
	for p := range matched {
		out = append(out, graph.ImporterID(p))
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out, nil
}

func expandFilterPattern(g *WorkspaceGraph, pat string) (map[string]struct{}, error) {
	pat = strings.TrimSpace(pat)
	if pat == "" {
		return nil, apperr.New(apperr.Usage, "workspace.filter", "", "empty filter pattern")
	}
	// ponytail: defer changed-since selectors ([origin/main]); upgrade when git integration lands.
	if strings.HasPrefix(pat, "[") {
		return nil, apperr.New(apperr.Usage, "workspace.filter", pat, "changed-since filter not implemented")
	}
	depsSuffix := strings.HasPrefix(pat, "...")
	dependentsSuffix := strings.HasSuffix(pat, "...")
	if depsSuffix {
		pat = strings.TrimPrefix(pat, "...")
	}
	if dependentsSuffix {
		pat = strings.TrimSuffix(pat, "...")
	}
	pats := expandBraces(pat)
	seed := make(map[string]struct{})
	for _, p := range pats {
		m, err := matchFilterPattern(g, p)
		if err != nil {
			return nil, err
		}
		for k := range m {
			seed[k] = struct{}{}
		}
	}
	if depsSuffix {
		seed = dependencyClosure(g, seed)
	}
	if dependentsSuffix {
		seed = dependentClosure(g, seed)
	}
	return seed, nil
}

func matchFilterPattern(g *WorkspaceGraph, pat string) (map[string]struct{}, error) {
	pat = strings.TrimSpace(pat)
	out := make(map[string]struct{})
	if pat == "" {
		return out, nil
	}
	if mem, ok := g.Members[pat]; ok {
		out[mem.Path] = struct{}{}
		return out, nil
	}
	for path := range g.ByPath {
		if filterPathMatch(pat, path) {
			out[path] = struct{}{}
		}
	}
	return out, nil
}

func filterPathMatch(pat, path string) bool {
	pat = filepath.ToSlash(strings.TrimSpace(pat))
	path = filepath.ToSlash(path)
	if pat == path {
		return true
	}
	if strings.ContainsAny(pat, "*?[") {
		ok, err := pathMatch(pat, path)
		return err == nil && ok
	}
	if strings.HasSuffix(path, pat) {
		return true
	}
	if strings.HasPrefix(path, pat) {
		return true
	}
	return false
}

func pathMatch(pat, path string) (bool, error) {
	parts := strings.Split(pat, "/")
	pathParts := strings.Split(path, "/")
	return matchPathParts(parts, pathParts)
}

func matchPathParts(patParts, pathParts []string) (bool, error) {
	if len(patParts) == 0 {
		return len(pathParts) == 0, nil
	}
	if patParts[0] == "**" {
		if len(patParts) == 1 {
			return true, nil
		}
		for i := 0; i <= len(pathParts); i++ {
			ok, err := matchPathParts(patParts[1:], pathParts[i:])
			if err != nil {
				return false, err
			}
			if ok {
				return true, nil
			}
		}
		return false, nil
	}
	if len(pathParts) == 0 {
		return false, nil
	}
	ok, err := filepath.Match(patParts[0], pathParts[0])
	if err != nil {
		return false, apperr.Wrap(apperr.Manifest, "workspace.filter", patParts[0], err)
	}
	if !ok {
		return false, nil
	}
	return matchPathParts(patParts[1:], pathParts[1:])
}

func dependencyClosure(g *WorkspaceGraph, seed map[string]struct{}) map[string]struct{} {
	out := make(map[string]struct{})
	for p := range seed {
		out[p] = struct{}{}
	}
	queue := make([]string, 0, len(seed))
	for p := range seed {
		queue = append(queue, p)
	}
	sort.Strings(queue)
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, depPath := range workspaceDepsOf(g, cur) {
			if _, ok := out[depPath]; ok {
				continue
			}
			out[depPath] = struct{}{}
			queue = append(queue, depPath)
		}
		sort.Strings(queue)
	}
	return out
}

func dependentClosure(g *WorkspaceGraph, seed map[string]struct{}) map[string]struct{} {
	out := make(map[string]struct{})
	for p := range seed {
		out[p] = struct{}{}
	}
	queue := make([]string, 0, len(seed))
	for p := range seed {
		queue = append(queue, p)
	}
	sort.Strings(queue)
	rev := reverseWorkspaceEdges(g)
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, dep := range rev[cur] {
			if _, ok := out[dep]; ok {
				continue
			}
			out[dep] = struct{}{}
			queue = append(queue, dep)
		}
		sort.Strings(queue)
	}
	return out
}

func reverseWorkspaceEdges(g *WorkspaceGraph) map[string][]string {
	rev := map[string][]string{}
	for path := range g.ByPath {
		for _, dep := range workspaceDepsOf(g, path) {
			rev[dep] = append(rev[dep], path)
		}
	}
	return rev
}

func workspaceDepsOf(g *WorkspaceGraph, path string) []string {
	docPath := filepath.Join(g.Root, filepath.FromSlash(path), "package.json")
	doc, err := manifest.Load(docPath)
	if err != nil {
		return nil
	}
	var out []string
	for _, depMap := range []map[string]string{doc.Dependencies, doc.DevDependencies, doc.OptionalDependencies} {
		for depName, spec := range depMap {
			mem, ok := WorkspaceTargetForDep(depName, spec, g)
			if !ok {
				continue
			}
			out = append(out, mem.Path)
		}
	}
	sort.Strings(out)
	return out
}
