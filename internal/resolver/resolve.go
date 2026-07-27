package resolver

import (
	"context"
	"fmt"
	"sort"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/config"
	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/manifest"
	"github.com/mewisme/m/internal/policy"
	"github.com/mewisme/m/internal/project"
	"github.com/mewisme/m/internal/registry"
)

// Engine resolves registry dependencies into a deterministic graph.
type Engine struct {
	Client    *registry.Client
	Effective *config.Effective
	Identity  project.Identity
}

// Ensure Engine implements Resolver.
var _ Resolver = (*Engine)(nil)

// NewEngine constructs an Engine.
func NewEngine(client *registry.Client, eff *config.Effective, identity project.Identity) *Engine {
	return &Engine{Client: client, Effective: eff, Identity: identity}
}

type workItem struct {
	from  string // importer id or package key
	name  string
	rng   string
	kind  graph.DepKind
	depth int
	path  []string // package names on the path from root (for cycles)
}

// ResolveProject expands an already-open project (used when manifest edits are in memory only).
func (e *Engine) ResolveProject(ctx context.Context, proj *project.Project, opts ResolveOptions) (*Resolution, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if e == nil || e.Client == nil {
		return nil, apperr.New(apperr.Internal, "resolver.resolve", "", "nil engine or client")
	}
	if proj == nil {
		return nil, apperr.New(apperr.Internal, "resolver.resolve", "", "nil project")
	}
	return e.resolveProject(ctx, proj, opts)
}

// Resolve expands the project at root into a complete Resolution.
func (e *Engine) Resolve(ctx context.Context, root string, opts ResolveOptions) (*Resolution, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if e == nil || e.Client == nil {
		return nil, apperr.New(apperr.Internal, "resolver.resolve", root, "nil engine or client")
	}
	proj, err := project.Open(ctx, root)
	if err != nil {
		return nil, err
	}
	return e.resolveProject(ctx, proj, opts)
}

func (e *Engine) resolveProject(ctx context.Context, proj *project.Project, opts ResolveOptions) (*Resolution, error) {
	pol := opts.Policy
	if pol == nil {
		pol = &policy.Policy{}
		_ = pol.Normalize()
	} else if err := pol.Normalize(); err != nil {
		return nil, err
	}
	hints := graphHints{g: opts.Hints}

	identity := e.Identity
	if identity == "" {
		identity = proj.Identity
	}

	b := graph.NewBuilder()
	b.Importer(graph.RootImporter, proj.Normalized.Name)

	decisions := []ResolutionDecision{}
	seenPkg := map[string]struct{}{}
	queuedEdge := map[string]struct{}{}

	var queue []workItem
	enqueue := func(from, name, rng string, kind graph.DepKind, depth int, path []string) {
		key := from + "\x00" + string(kind) + "\x00" + name + "\x00" + rng
		if _, ok := queuedEdge[key]; ok {
			return
		}
		queuedEdge[key] = struct{}{}
		queue = append(queue, workItem{from: from, name: name, rng: rng, kind: kind, depth: depth, path: path})
	}

	seedDeps(proj.Normalized, opts.OmitRootDev, func(name, rng string, kind graph.DepKind) {
		enqueue(string(graph.RootImporter), name, rng, kind, 1, nil)
	})

	for len(queue) > 0 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		sort.SliceStable(queue, func(i, j int) bool {
			a, c := queue[i], queue[j]
			if a.name != c.name {
				return a.name < c.name
			}
			if a.kind != c.kind {
				return a.kind < c.kind
			}
			if a.rng != c.rng {
				return a.rng < c.rng
			}
			return a.from < c.from
		})
		item := queue[0]
		queue = queue[1:]

		if item.depth > maxDepth {
			return nil, apperr.New(apperr.Resolve, "resolver.limit", item.name,
				fmt.Sprintf("resolution depth exceeded %d", maxDepth))
		}
		if pathContains(item.path, item.name) {
			return nil, cycleError(item.path, item.name)
		}

		base := registry.ResolveBaseForPackage(e.Effective, proj.Root, identity, item.name)
		pack, err := e.Client.Packument(ctx, base, item.name)
		if err != nil {
			return nil, apperr.Wrap(apperr.Resolve, "resolver.packument", item.name, err)
		}

		meta, decision, err := selectVersion(pack, item.name, item.rng, pol, &hints)
		if err != nil {
			return nil, err
		}
		decisions = append(decisions, decision)

		id := graph.PackageID{Name: item.name, Version: meta.Version}
		key := id.Key()
		tarball := registry.AbsoluteTarballURL(base, item.name, meta.Dist.Tarball)
		if tarball == "" {
			tarball = meta.Dist.Tarball
		}

		b.Edge(item.from, key, item.kind, item.rng)

		if _, ok := seenPkg[key]; ok {
			continue
		}
		if len(seenPkg) >= maxPackages {
			return nil, apperr.New(apperr.Resolve, "resolver.limit", item.name,
				fmt.Sprintf("resolution package count exceeded %d", maxPackages))
		}
		seenPkg[key] = struct{}{}
		b.Package(id, meta.Dist.Integrity, tarball)

		nextPath := append(append([]string(nil), item.path...), item.name)
		// Expand prod dependencies only — never transitive devDependencies.
		for _, kv := range sortedDeps(meta.Dependencies) {
			enqueue(key, kv.name, kv.rng, graph.DepProd, item.depth+1, nextPath)
		}
	}

	g, err := b.Build()
	if err != nil {
		return nil, apperr.Wrap(apperr.Resolve, "resolver.build", proj.Root, err)
	}
	return &Resolution{
		SchemaVersion: ResolutionSchemaVersion,
		Graph:         g,
		Decisions:     decisions,
	}, nil
}

func seedDeps(m *manifest.Manifest, omitDev bool, add func(name, rng string, kind graph.DepKind)) {
	if m == nil {
		return
	}
	for _, d := range m.Dependencies {
		switch d.Kind {
		case manifest.DepProd:
			add(d.Name, d.Range, d.Kind)
		case manifest.DepDev:
			if !omitDev {
				add(d.Name, d.Range, d.Kind)
			}
		}
	}
}

type namedRange struct {
	name, rng string
}

func sortedDeps(m map[string]string) []namedRange {
	if len(m) == 0 {
		return nil
	}
	out := make([]namedRange, 0, len(m))
	for k, v := range m {
		out = append(out, namedRange{name: k, rng: v})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}
