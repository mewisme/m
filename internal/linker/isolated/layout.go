package isolated

import (
	"path/filepath"
	"sort"

	"github.com/mewisme/m/internal/graph"
)

const virtualStoreDir = ".pnpm"

type pkgLayout struct {
	Key        string
	StoreID    string
	ContentDir string // absolute path under nmRoot
	PrivateNM  string // .pnpm/<id>/node_modules
}

type aliasLink struct {
	Src  string // target package content dir
	Dest string // symlink/junction destination
}

type depEdge struct {
	name  string
	toKey string
}

// Layout describes isolated virtual store paths for g under nmRoot.
type Layout struct {
	Packages []pkgLayout
	Aliases  []aliasLink
	DepLinks []aliasLink // internal package node_modules links
}

func computeLayout(g *graph.Graph, nmRoot string) (*Layout, error) {
	if g == nil {
		return &Layout{}, nil
	}
	if err := CheckStoreIDCollisions(g.Packages); err != nil {
		return nil, err
	}
	byKey := map[string]graph.Package{}
	for _, p := range g.Packages {
		byKey[p.ID.Key()] = p
	}
	out := &Layout{}
	seen := map[string]struct{}{}
	for _, p := range g.Packages {
		key := p.ID.Key()
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		sid := StoreID(p.ID)
		privateNM := filepath.Join(nmRoot, virtualStoreDir, sid, "node_modules")
		contentDir := filepath.Join(append([]string{privateNM}, installSegments(p.ID.Name)...)...)
		out.Packages = append(out.Packages, pkgLayout{
			Key: key, StoreID: sid, ContentDir: contentDir, PrivateNM: privateNM,
		})
	}
	sort.Slice(out.Packages, func(i, j int) bool {
		return out.Packages[i].StoreID < out.Packages[j].StoreID
	})

	children := childDepEdges(g)
	contentOf := map[string]string{}
	for _, pl := range out.Packages {
		contentOf[pl.Key] = pl.ContentDir
	}

	for from, edges := range children {
		for _, edge := range edges {
			target, ok := contentOf[edge.toKey]
			if !ok {
				continue
			}
			if from == string(graph.RootImporter) {
				dest := filepath.Join(append([]string{nmRoot}, installSegments(edge.name)...)...)
				out.Aliases = append(out.Aliases, aliasLink{Src: target, Dest: dest})
				continue
			}
			parentPrivate := privateNMFor(out.Packages, from)
			if parentPrivate == "" {
				continue
			}
			dest := filepath.Join(append([]string{parentPrivate}, installSegments(edge.name)...)...)
			out.DepLinks = append(out.DepLinks, aliasLink{Src: target, Dest: dest})
		}
	}
	sort.Slice(out.Aliases, func(i, j int) bool { return out.Aliases[i].Dest < out.Aliases[j].Dest })
	sort.Slice(out.DepLinks, func(i, j int) bool { return out.DepLinks[i].Dest < out.DepLinks[j].Dest })
	return out, nil
}

// PackageContentDirs returns package key -> content directory for tests and validation.
func PackageContentDirs(g *graph.Graph, nmRoot string) (map[string]string, error) {
	layout, err := computeLayout(g, nmRoot)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(layout.Packages))
	for _, p := range layout.Packages {
		out[p.Key] = p.ContentDir
	}
	return out, nil
}

func privateNMFor(pkgs []pkgLayout, key string) string {
	for _, p := range pkgs {
		if p.Key == key {
			return p.PrivateNM
		}
	}
	return ""
}

func childDepEdges(g *graph.Graph) map[string][]depEdge {
	out := make(map[string][]depEdge)
	for _, e := range g.Edges {
		name := e.Name
		if name == "" {
			name = graph.TargetNameFromKey(e.To)
		}
		out[e.From] = append(out[e.From], depEdge{name: name, toKey: e.To})
	}
	for from := range out {
		sort.Slice(out[from], func(i, j int) bool {
			a, b := out[from][i], out[from][j]
			if a.name != b.name {
				return a.name < b.name
			}
			return a.toKey < b.toKey
		})
	}
	return out
}

func installSegments(name string) []string {
	if len(name) > 0 && name[0] == '@' {
		rest := name[1:]
		for i := 0; i < len(rest); i++ {
			if rest[i] == '/' {
				return []string{"@" + rest[:i], rest[i+1:]}
			}
		}
		return []string{"@" + rest}
	}
	return []string{name}
}
