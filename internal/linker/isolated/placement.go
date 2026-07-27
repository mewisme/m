package isolated

import (
	"context"
	"path/filepath"
	"sort"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/linker"
	"github.com/mewisme/m/internal/linker/planner"
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
	rootDeps := rootDependencyKeys(g)

	for _, pl := range layout.Packages {
		src, ok := extracts[pl.Key]
		if !ok || src == "" {
			return nil, apperr.New(apperr.Internal, "linker.isolated.plan", pl.Key, "missing extract dir")
		}
		ops = append(ops, linker.Op{Kind: linker.OpMkdir, Dest: pl.ContentDir})
		if l.UseSmartLink {
			ops = append(ops, planner.PlanPackageLink(src, pl.ContentDir, caps))
		} else {
			ops = append(ops, linker.Op{Kind: linker.OpCopy, Src: src, Dest: pl.ContentDir})
		}
		placements = append(placements, linker.Placement{Key: pl.Key, DestDir: pl.ContentDir})

		if rootDeps[pl.Key] {
			aliasDest := filepath.Join(append([]string{nmRoot}, installSegments(packageNameFromKey(pl.Key))...)...)
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
	for _, link := range layout.DepLinks {
		ops = append(ops, linker.Op{Kind: linker.OpMkdir, Dest: filepath.Dir(link.Dest)})
		ops = append(ops, planner.PlanDirAlias(link.Src, link.Dest, caps))
	}
	for _, link := range layout.Aliases {
		ops = append(ops, linker.Op{Kind: linker.OpMkdir, Dest: filepath.Dir(link.Dest)})
		ops = append(ops, planner.PlanDirAlias(link.Src, link.Dest, caps))
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
