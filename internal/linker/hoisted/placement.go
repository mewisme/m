package hoisted

import (
	"context"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/linker"
	"github.com/mewisme/m/internal/linker/planner"
)

// Linker plans and applies a conservative copy-based hoisted layout.
type Linker struct {
	NodeModules  string
	ExtractDirs  map[string]string
	Capabilities planner.Capabilities
	UseSmartLink bool
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

	rootDirect := rootDependencyKeys(g)
	seenDest := make(map[string]struct{}, len(placements))
	ops := make([]linker.Op, 0, len(placements)*2)
	bins := make([]linker.BinSource, 0)
	for _, p := range placements {
		src, ok := extracts[p.Key]
		if !ok || src == "" {
			return nil, apperr.New(apperr.Internal, "linker.hoisted.plan", p.Key, "missing extract dir")
		}
		if _, dup := seenDest[p.DestDir]; !dup {
			seenDest[p.DestDir] = struct{}{}
			ops = append(ops, linker.Op{Kind: linker.OpMkdir, Dest: p.DestDir})
			if l.UseSmartLink {
				ops = append(ops, planner.PlanPackageLink(src, p.DestDir, l.Capabilities))
			} else {
				ops = append(ops, linker.Op{Kind: linker.OpCopy, Src: src, Dest: p.DestDir})
			}
		}
		cmds, err := linker.BinCommandsFromDirNamed(src, packageName(p.Key))
		if err != nil {
			return nil, err
		}
		var binNM string
		switch {
		case p.ID.HoistLevel > 0:
			binNM = binNodeModulesFor(p.DestDir)
		case rootDirect[p.Key]:
			binNM = nmRoot
		default:
			continue
		}
		if binNM == "" {
			continue
		}
		for cmd, target := range cmds {
			bins = append(bins, linker.BinSource{
				Cmd:         cmd,
				Target:      target,
				PackageDir:  p.DestDir,
				NodeModules: binNM,
			})
		}
	}
	sort.SliceStable(bins, func(i, j int) bool {
		if bins[i].NodeModules != bins[j].NodeModules {
			return bins[i].NodeModules < bins[j].NodeModules
		}
		return bins[i].Cmd < bins[j].Cmd
	})

	return &linker.Plan{
		LayoutMode:  "hoisted",
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

type walkCtx struct {
	id       linker.PlacementID
	destDir  string
	hoistLvl int
}

type hoistEntry struct {
	version string
	key     string
}

type depEdge struct {
	name  string
	toKey string
}

func placements(g *graph.Graph, nmRoot string) ([]linker.Placement, error) {
	children := childDepEdges(g)
	projectRoot := filepath.Dir(nmRoot)
	rootHoisted := map[string]hoistEntry{}
	for _, e := range children[string(graph.RootImporter)] {
		rootHoisted[e.name] = hoistEntry{
			version: packageVersion(e.toKey),
			key:     e.toKey,
		}
	}

	var result []linker.Placement
	visitedEdges := map[string]struct{}{}
	destCtx := map[string]walkCtx{}

	var walk func(parent walkCtx, fromKey string, parentPkgDir string)
	walk = func(parent walkCtx, fromKey string, parentPkgDir string) {
		importer := fromKey
		if fromKey == string(graph.RootImporter) {
			importer = string(graph.RootImporter)
		}
		for _, edge := range children[fromKey] {
			edgeKey := fromKey + "->" + edge.name + "->" + edge.toKey
			if _, seen := visitedEdges[edgeKey]; seen {
				continue
			}
			visitedEdges[edgeKey] = struct{}{}

			depName := edge.name
			version := packageVersion(edge.toKey)
			fromRoot := fromKey == string(graph.RootImporter)
			destDir, hoistLvl := resolveHoistedDest(fromRoot, parentPkgDir, nmRoot, edge.toKey, depName, version, rootHoisted)

			if existing, ok := destCtx[destDir]; ok && existing.destDir == destDir {
				walk(existing, edge.toKey, destDir)
				continue
			}

			pid := linker.PlacementID{
				Parent:      parent.id.String(),
				Importer:    importer,
				DepName:     depName,
				PackageKey:  edge.toKey,
				HoistLevel:  hoistLvl,
				PeerContext: peerContextDigest(edge.toKey),
			}
			ctx := walkCtx{id: pid, destDir: destDir, hoistLvl: hoistLvl}
			destCtx[destDir] = ctx
			result = append(result, linker.Placement{ID: pid, Key: edge.toKey, DestDir: destDir})
			walk(ctx, edge.toKey, destDir)
		}
	}

	walk(walkCtx{}, string(graph.RootImporter), projectRoot)

	sort.SliceStable(result, func(i, j int) bool {
		return result[i].ID.Compare(result[j].ID) < 0
	})
	return result, nil
}

func resolveHoistedDest(fromRoot bool, parentPkgDir, nmRoot, packageKey, edgeName, version string, rootHoisted map[string]hoistEntry) (destDir string, hoistLvl int) {
	if fromRoot {
		entry, exists := rootHoisted[edgeName]
		if !exists {
			rootHoisted[edgeName] = hoistEntry{version: version, key: packageKey}
			return installUnder(nmRoot, edgeName), 0
		}
		if entry.version == version && entry.key == packageKey {
			return installUnder(nmRoot, edgeName), 0
		}
		nested := filepath.Join(parentPkgDir, "node_modules")
		return installUnder(nested, edgeName), hoistLevelFor(nested, nmRoot)
	}
	if entry, exists := rootHoisted[edgeName]; exists {
		if entry.version == version && entry.key == packageKey {
			return installUnder(nmRoot, edgeName), 0
		}
		nested := filepath.Join(parentPkgDir, "node_modules")
		return installUnder(nested, edgeName), hoistLevelFor(nested, nmRoot)
	}
	rootHoisted[edgeName] = hoistEntry{version: version, key: packageKey}
	return installUnder(nmRoot, edgeName), 0
}

func peerContextDigest(packageKey string) string {
	if i := strings.IndexByte(packageKey, '#'); i >= 0 {
		return packageKey[i+1:]
	}
	return ""
}

func hoistLevelFor(scopeDir, nmRoot string) int {
	scopeDir = filepath.Clean(scopeDir)
	nmRoot = filepath.Clean(nmRoot)
	if scopeDir == nmRoot {
		return 0
	}
	rel, err := filepath.Rel(nmRoot, scopeDir)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return 1
	}
	return strings.Count(filepath.ToSlash(rel), "node_modules")
}

func binNodeModulesFor(pkgInstallDir string) string {
	dir := filepath.Clean(pkgInstallDir)
	for {
		parent := filepath.Dir(dir)
		if filepath.Base(parent) == "node_modules" {
			return parent
		}
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func rootDependencyKeys(g *graph.Graph) map[string]bool {
	out := map[string]bool{}
	for _, e := range g.Edges {
		if e.From == string(graph.RootImporter) {
			out[e.To] = true
		}
	}
	return out
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

func packageName(key string) string {
	return graph.TargetNameFromKey(key)
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
