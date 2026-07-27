package isolated

import (
	"path/filepath"
	"sort"
	"strings"

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

	children := childEdges(g)
	contentOf := map[string]string{}
	for _, pl := range out.Packages {
		contentOf[pl.Key] = pl.ContentDir
	}

	for from, tos := range children {
		for _, toKey := range tos {
			childName := packageNameFromKey(toKey)
			target, ok := contentOf[toKey]
			if !ok {
				continue
			}
			if from == string(graph.RootImporter) {
				dest := filepath.Join(append([]string{nmRoot}, installSegments(childName)...)...)
				out.Aliases = append(out.Aliases, aliasLink{Src: target, Dest: dest})
				continue
			}
			parentPrivate := privateNMFor(out.Packages, from)
			if parentPrivate == "" {
				continue
			}
			dest := filepath.Join(append([]string{parentPrivate}, installSegments(childName)...)...)
			out.DepLinks = append(out.DepLinks, aliasLink{Src: target, Dest: dest})
		}
	}
	sort.Slice(out.Aliases, func(i, j int) bool { return out.Aliases[i].Dest < out.Aliases[j].Dest })
	sort.Slice(out.DepLinks, func(i, j int) bool { return out.DepLinks[i].Dest < out.DepLinks[j].Dest })
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

func childEdges(g *graph.Graph) map[string][]string {
	out := make(map[string][]string)
	for _, e := range g.Edges {
		out[e.From] = append(out[e.From], e.To)
	}
	for from := range out {
		sort.Strings(out[from])
	}
	return out
}

func installSegments(name string) []string {
	if strings.HasPrefix(name, "@") {
		rest := name[1:]
		if i := strings.IndexByte(rest, '/'); i >= 0 {
			return []string{"@" + rest[:i], rest[i+1:]}
		}
		return []string{"@" + rest}
	}
	return []string{name}
}
