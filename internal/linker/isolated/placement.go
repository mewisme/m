package isolated

import (
	"context"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/linker"
	"github.com/mewisme/mew/internal/linker/planner"
)

// Linker plans and applies a pnpm-style isolated virtual store layout.
type Linker struct {
	NodeModules  string
	ExtractDirs  map[string]string
	Capabilities planner.Capabilities
	UseSmartLink bool
}

// Plan computes virtual store ops for g.
func (l *Linker) Plan(ctx context.Context, g *graph.Graph) (*linker.Plan, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if l == nil {
		return nil, apperr.New(apperr.Internal, "linker.isolated.plan", "linker", "nil linker")
	}
	if g == nil {
		return nil, apperr.New(apperr.Internal, "linker.isolated.plan", "graph", "nil graph")
	}
	if l.NodeModules == "" {
		return nil, apperr.New(apperr.Internal, "linker.isolated.plan", "nodeModules", "empty path")
	}
	nmRoot, err := filepath.Abs(l.NodeModules)
	if err != nil {
		return nil, apperr.Wrap(apperr.IO, "linker.isolated.plan", l.NodeModules, err)
	}
	layout, err := computeLayout(g, nmRoot)
	if err != nil {
		return nil, err
	}
	extracts := l.ExtractDirs
	if extracts == nil {
		extracts = map[string]string{}
	}
	caps := l.Capabilities

	ops := make([]linker.Op, 0)
	placements := make([]linker.Placement, 0, len(layout.Packages))
	bins := make([]linker.BinSource, 0)
	seenAliasDest := map[string]struct{}{}
	rootDeps := rootDependencyKeys(g)

	for _, pl := range layout.Packages {
		if strings.HasPrefix(pl.Key, "link:") {
			continue
		}
		src, ok := extracts[pl.Key]
		if !ok || src == "" {
			return nil, apperr.New(apperr.Internal, "linker.isolated.plan", pl.Key, "missing extract dir")
		}
		ops = append(ops, linker.Op{Kind: linker.OpMkdir, Dest: pl.ContentDir})
		switch {
		case l.UseSmartLink:
			ops = append(ops, planner.PlanPackageLink(src, pl.ContentDir, caps))
		case caps.Junction || caps.Symlink:
			ops = append(ops, planner.PlanDirAlias(src, pl.ContentDir, caps))
		default:
			ops = append(ops, linker.Op{Kind: linker.OpCopy, Src: src, Dest: pl.ContentDir})
		}
		placements = append(placements, linker.Placement{Key: pl.Key, DestDir: pl.ContentDir})

		if rootDeps[pl.Key] {
			edgeName := packageNameFromKey(pl.Key)
			for _, e := range g.Edges {
				if e.From == string(graph.RootImporter) && e.To == pl.Key {
					if e.Name != "" {
						edgeName = e.Name
					}
					break
				}
			}
			aliasDest := filepath.Join(append([]string{nmRoot}, installSegments(edgeName)...)...)
			cmds, err := linker.BinCommandsFromDir(src)
			if err != nil {
				return nil, err
			}
			for cmd, target := range cmds {
				bins = append(bins, linker.BinSource{
					Cmd: cmd, Target: target, PackageDir: aliasDest,
				})
			}
		}
	}
	appendAlias := func(src, dest string) {
		dest = filepath.Clean(dest)
		if _, dup := seenAliasDest[dest]; dup {
			return
		}
		seenAliasDest[dest] = struct{}{}
		ops = append(ops, linker.Op{Kind: linker.OpMkdir, Dest: filepath.Dir(dest)})
		ops = append(ops, planner.PlanDirAlias(src, dest, caps))
	}
	for _, link := range layout.DepLinks {
		appendAlias(link.Src, link.Dest)
	}
	for _, link := range layout.Aliases {
		appendAlias(link.Src, link.Dest)
	}
	for _, e := range g.Edges {
		if e.From != string(graph.RootImporter) || !strings.HasPrefix(e.To, "link:") {
			continue
		}
		src, ok := extracts[e.To]
		if !ok || src == "" {
			return nil, apperr.New(apperr.Internal, "linker.isolated.plan", e.To, "missing extract dir for link protocol")
		}
		name := e.Name
		if name == "" {
			name = graph.TargetNameFromKey(e.To)
		}
		dest := filepath.Join(append([]string{nmRoot}, installSegments(name)...)...)
		appendAlias(src, dest)
	}
	sort.SliceStable(bins, func(i, j int) bool { return bins[i].Cmd < bins[j].Cmd })

	return &linker.Plan{
		LayoutMode:  "isolated",
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

func rootDependencyKeys(g *graph.Graph) map[string]bool {
	out := map[string]bool{}
	for _, e := range g.Edges {
		if e.From == string(graph.RootImporter) {
			out[e.To] = true
		}
	}
	return out
}
