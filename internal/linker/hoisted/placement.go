package hoisted

import (
	"context"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/linker"
)

// Linker plans and applies a conservative copy-based hoisted layout.
type Linker struct {
	NodeModules string
	ExtractDirs map[string]string
}

// Plan computes placements and copy ops for g.
func (l *Linker) Plan(ctx context.Context, g *graph.Graph) (*linker.Plan, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if l == nil {
		return nil, apperr.New(apperr.Internal, "linker.hoisted.plan", "linker", "nil linker")
	}
	if g == nil {
		return nil, apperr.New(apperr.Internal, "linker.hoisted.plan", "graph", "nil graph")
	}
	if l.NodeModules == "" {
		return nil, apperr.New(apperr.Internal, "linker.hoisted.plan", "nodeModules", "empty path")
	}
	nmRoot, err := filepath.Abs(l.NodeModules)
	if err != nil {
		return nil, apperr.Wrap(apperr.IO, "linker.hoisted.plan", l.NodeModules, err)
	}

	placements, err := placements(g, nmRoot)
	if err != nil {
		return nil, err
	}

	extracts := l.ExtractDirs
	if extracts == nil {
		extracts = map[string]string{}
	}

	ops := make([]linker.Op, 0, len(placements)*2)
	bins := make([]linker.BinSource, 0)
	for _, p := range placements {
		src, ok := extracts[p.Key]
		if !ok || src == "" {
			return nil, apperr.New(apperr.Internal, "linker.hoisted.plan", p.Key, "missing extract dir")
		}
		ops = append(ops,
			linker.Op{Kind: linker.OpMkdir, Dest: p.DestDir},
			linker.Op{Kind: linker.OpCopy, Src: src, Dest: p.DestDir},
		)
		cmds, err := linker.BinCommandsFromDir(src)
		if err != nil {
			return nil, err
		}
		for cmd, target := range cmds {
			bins = append(bins, linker.BinSource{
				Cmd:        cmd,
				Target:     target,
				PackageDir: p.DestDir,
			})
		}
	}
	sort.SliceStable(bins, func(i, j int) bool {
		return bins[i].Cmd < bins[j].Cmd
	})

	return &linker.Plan{
		NodeModules: nmRoot,
		ExtractDirs: extracts,
		Placements:  placements,
		Ops:         ops,
		Bins:        bins,
	}, nil
}

// Apply executes a plan produced by Plan.
func (l *Linker) Apply(ctx context.Context, plan *linker.Plan) error {
	return linker.Apply(ctx, plan)
}

var _ linker.Linker = (*Linker)(nil)

// Placements computes install directories for packages in g under nmRoot.
func Placements(g *graph.Graph, nmRoot string) ([]linker.Placement, error) {
	return placements(g, nmRoot)
}

func placements(g *graph.Graph, nmRoot string) ([]linker.Placement, error) {
	children := childEdges(g)
	projectRoot := filepath.Dir(nmRoot)

	hoisted := make(map[string]string) // name -> version at root
	placed := make(map[string]string)  // key -> dest dir
	placementOrder := make([]linker.Placement, 0, len(g.Packages))

	var walk func(from string, parentDir string)
	walk = func(from string, parentDir string) {
		for _, toKey := range children[from] {
			if dest, ok := placed[toKey]; ok {
				walk(toKey, dest)
				continue
			}
			name := packageName(toKey)
			version := packageVersion(toKey)
			var destDir string
			if rootVer, exists := hoisted[name]; exists {
				if rootVer == version {
					destDir = installUnder(nmRoot, name)
				} else {
					destDir = installUnder(filepath.Join(parentDir, "node_modules"), name)
				}
			} else {
				hoisted[name] = version
				destDir = installUnder(nmRoot, name)
			}
			placed[toKey] = destDir
			placementOrder = append(placementOrder, linker.Placement{Key: toKey, DestDir: destDir})
			walk(toKey, destDir)
		}
	}

	walk(string(graph.RootImporter), projectRoot)
	return placementOrder, nil
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

func packageName(key string) string {
	s := key
	if i := strings.IndexByte(s, '#'); i >= 0 {
		s = s[:i]
	}
	if i := strings.LastIndexByte(s, '@'); i > 0 {
		return s[:i]
	}
	return s
}

func packageVersion(key string) string {
	s := key
	if i := strings.IndexByte(s, '#'); i >= 0 {
		s = s[:i]
	}
	if i := strings.LastIndexByte(s, '@'); i > 0 {
		return s[i+1:]
	}
	return ""
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

func installUnder(root, name string) string {
	return filepath.Join(append([]string{root}, installSegments(name)...)...)
}
