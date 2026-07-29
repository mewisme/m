package resolver

import (
	"sort"
	"strings"

	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/semver"
)

// consolidateDuplicateNames collapses multiple versions of the same package name in g
// when every edge range referencing that name allows a single shared version.
//
// ponytail: O(n²) name scan; upgrade path = full SAT-style dedupe later.
func consolidateDuplicateNames(g *graph.Graph, prior *graph.Graph) (*graph.Graph, error) {
	if g == nil || prior == nil {
		return g, nil
	}
	dupNames := duplicateNamesInGraph(prior)
	if len(dupNames) == 0 {
		return g, nil
	}
	out := g
	for _, name := range dupNames {
		target, ok := pickConsolidationVersion(out, name)
		if !ok {
			continue
		}
		remapped, err := remapPackageNameVersion(out, name, target)
		if err != nil {
			return nil, err
		}
		out = remapped
	}
	return out, nil
}

func duplicateNamesInGraph(g *graph.Graph) []string {
	counts := map[string]int{}
	for _, p := range g.Packages {
		counts[p.ID.Name]++
	}
	var names []string
	for name, n := range counts {
		if n > 1 {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func pickConsolidationVersion(g *graph.Graph, name string) (string, bool) {
	if g == nil {
		return "", false
	}
	versions := map[string]struct{}{}
	for _, p := range g.Packages {
		if p.ID.Name == name {
			versions[p.ID.Version] = struct{}{}
		}
	}
	if len(versions) <= 1 {
		return "", false
	}
	ranges := edgeRangesForName(g, name)
	if len(ranges) == 0 {
		return "", false
	}
	var candidates []string
	for ver := range versions {
		ok := true
		for _, rng := range ranges {
			sat, err := semver.Satisfies(ver, rng)
			if err != nil || !sat {
				ok = false
				break
			}
		}
		if ok {
			candidates = append(candidates, ver)
		}
	}
	if len(candidates) == 0 {
		return "", false
	}
	sort.Slice(candidates, func(i, j int) bool {
		cmp, _ := semver.Compare(candidates[i], candidates[j])
		return cmp > 0
	})
	return candidates[0], true
}

func edgeRangesForName(g *graph.Graph, name string) []string {
	seen := map[string]struct{}{}
	var ranges []string
	for _, e := range g.Edges {
		edgeName := e.Name
		if edgeName == "" {
			edgeName = graph.TargetNameFromKey(e.To)
		}
		if edgeName != name {
			continue
		}
		if e.Range == "" {
			continue
		}
		if _, ok := seen[e.Range]; ok {
			continue
		}
		seen[e.Range] = struct{}{}
		ranges = append(ranges, e.Range)
	}
	sort.Strings(ranges)
	return ranges
}

func remapPackageNameVersion(g *graph.Graph, name, targetVersion string) (*graph.Graph, error) {
	if g == nil {
		return nil, nil
	}
	targetKey := graph.PackageID{Name: name, Version: targetVersion}.Key()
	b := graph.NewBuilder()
	for _, im := range g.Importers {
		b.Importer(im.ID, im.Name)
	}
	for _, p := range g.Packages {
		if p.ID.Name == name && p.ID.Version != targetVersion {
			continue
		}
		b.Package(p.ID, p.Integrity, p.TarballURL)
	}
	for _, e := range g.Edges {
		to := e.To
		if graph.TargetNameFromKey(e.To) == name && packageKeyVersion(e.To) != targetVersion {
			to = remapKeyPeerContext(e.To, targetKey)
		}
		b.EdgeEx(e.From, e.Name, to, e.Kind, e.Range, e.Optional)
	}
	return b.Build()
}

func packageKeyVersion(key string) string {
	at := strings.IndexByte(key, '@')
	if at < 0 {
		return ""
	}
	rest := key[at+1:]
	if i := strings.IndexByte(rest, '#'); i >= 0 {
		rest = rest[:i]
	}
	if i := strings.IndexByte(rest, '('); i >= 0 {
		rest = rest[:i]
	}
	return rest
}

func remapKeyPeerContext(oldKey, newKey string) string {
	if i := strings.IndexByte(oldKey, '#'); i >= 0 {
		if strings.IndexByte(newKey, '#') >= 0 {
			return newKey
		}
		return newKey + oldKey[i:]
	}
	return newKey
}
