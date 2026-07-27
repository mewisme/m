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
	"github.com/mewisme/m/internal/workspace"
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
	from         string
	display      string
	name         string
	spec         string
	rng          string
	protocol     manifest.Protocol
	declarerPath string
	kind         graph.DepKind
	depth        int
	path         []string // package name path for overrides
	envKeys      []string // ancestor package keys for peer resolution
	optional     bool
	overrideFrom string
}

type resolveState struct {
	e           *Engine
	ctx         context.Context
	proj        *project.Project
	pol         *policy.Policy
	hints       graphHints
	target      Target
	overrides   map[string]string
	identity    project.Identity
	omitRootDev bool

	b            *graph.Builder
	decisions    []ResolutionDecision
	seenPkg      map[string]struct{}
	resolving    map[string]struct{}
	queuedEdge   map[string]struct{}
	pkgPeers     map[string]map[string]string
	pkgPeerOpt   map[string]map[string]bool
	pkgEnv       map[string][]string
	pkgFrom      map[string]string
	provides     map[string]map[string]providedDep
	pkgInstances map[string]string // instanceKey -> current graph package key
	queue        []workItem

	wsIndex         *workspace.Index
	wsByName        map[string]workspaceMember
	wsMemberPaths   map[string]struct{}
	localSources    map[string]LocalSource
	seededImporters map[graph.ImporterID]bool
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
		pol = PolicyFromEffective(e.Effective)
	}
	if err := pol.Normalize(); err != nil {
		return nil, err
	}

	identity := e.Identity
	if identity == "" {
		identity = proj.Identity
	}

	s := &resolveState{
		e:               e,
		ctx:             ctx,
		proj:            proj,
		pol:             pol,
		hints:           prepareHints(e.Effective, opts, proj.Normalized),
		target:          CurrentTarget(),
		overrides:       proj.Normalized.Overrides,
		identity:        identity,
		omitRootDev:     opts.OmitRootDev,
		b:               graph.NewBuilder(),
		seenPkg:         map[string]struct{}{},
		resolving:       map[string]struct{}{},
		queuedEdge:      map[string]struct{}{},
		pkgPeers:        map[string]map[string]string{},
		pkgPeerOpt:      map[string]map[string]bool{},
		pkgEnv:          map[string][]string{},
		pkgFrom:         map[string]string{},
		provides:        map[string]map[string]providedDep{},
		pkgInstances:    map[string]string{},
		localSources:    map[string]LocalSource{},
		seededImporters: map[graph.ImporterID]bool{graph.RootImporter: true},
	}
	s.b.Importer(graph.RootImporter, proj.Normalized.Name)

	if err := s.initWorkspace(); err != nil {
		return nil, err
	}
	if err := s.seedFromManifest(proj.Normalized); err != nil {
		return nil, err
	}
	if err := s.seedWorkspaceMembers(); err != nil {
		return nil, err
	}
	if err := s.run(); err != nil {
		return nil, err
	}
	s.finalizePeerProviderContexts()
	if err := s.validatePeers(pol); err != nil {
		return nil, err
	}
	if err := s.canonicalizePeerInstances(); err != nil {
		return nil, err
	}

	g, err := s.b.Build()
	if err != nil {
		return nil, apperr.Wrap(apperr.Resolve, "resolver.build", proj.Root, err)
	}
	if s.hints.incremental && opts.Prior != nil {
		g, err = mergeUnchangedSubgraph(opts.Prior, g, s.hints.updateClosure)
		if err != nil {
			return nil, apperr.Wrap(apperr.Resolve, "resolver.merge", proj.Root, err)
		}
	}
	return &Resolution{
		SchemaVersion: ResolutionSchemaVersion,
		Graph:         g,
		Decisions:     s.decisions,
		Extensions:    s.buildExtensions(),
	}, nil
}

func (s *resolveState) seedFromManifest(m *manifest.Manifest) error {
	return s.seedDeps(string(graph.RootImporter), ".", m, 1, nil, nil)
}

func (s *resolveState) seedDeps(from, declarerPath string, m *manifest.Manifest, depth int, namePath, envKeys []string) error {
	if m == nil {
		return nil
	}
	omitDev := from == string(graph.RootImporter) && s.omitRootDev
	for _, d := range m.Dependencies {
		if d.Kind == manifest.DepDev && omitDev {
			continue
		}
		kind := d.Kind
		optional := false
		switch d.Kind {
		case manifest.DepPeer:
			continue
		case manifest.DepOptional:
			kind = graph.DepOptional
			optional = true
		}
		if err := s.enqueue(from, declarerPath, d.Name, d.Range, kind, depth, namePath, envKeys, optional); err != nil {
			return err
		}
	}
	return nil
}

func (s *resolveState) enqueue(from, declarerPath, display, spec string, kind graph.DepKind, depth int, namePath, envKeys []string, optional bool) error {
	overrideFrom := ""
	if _, ok := matchOverride(s.overrides, namePath, display); ok {
		overrideFrom = spec
	}
	display, target, rng, protocol, err := rewriteSpecifier(s.overrides, namePath, display, spec)
	if err != nil {
		return err
	}
	if declarerPath == "" {
		declarerPath = s.declarerPathFor(from)
	}
	key := from + "\x00" + string(kind) + "\x00" + target + "\x00" + spec
	if _, ok := s.queuedEdge[key]; ok {
		return nil
	}
	s.queuedEdge[key] = struct{}{}
	s.queue = append(s.queue, workItem{
		from: from, display: display, name: target, spec: spec, rng: rng,
		protocol: protocol, declarerPath: declarerPath,
		kind: kind, depth: depth, path: namePath, envKeys: envKeys, optional: optional,
		overrideFrom: overrideFrom,
	})
	return nil
}

func (s *resolveState) declarerPathFor(from string) string {
	if from == string(graph.RootImporter) {
		return "."
	}
	if s.wsMemberPaths != nil {
		if _, ok := s.wsMemberPaths[from]; ok {
			return from
		}
	}
	if loc, ok := s.localSources[from]; ok {
		return loc.Path
	}
	id := parsePackageKey(from)
	if loc, ok := s.localSources[id.Key()]; ok {
		return loc.Path
	}
	return ""
}

func (s *resolveState) run() error {
	for len(s.queue) > 0 {
		if err := s.ctx.Err(); err != nil {
			return err
		}
		sort.SliceStable(s.queue, func(i, j int) bool {
			a, c := s.queue[i], s.queue[j]
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
		item := s.queue[0]
		s.queue = s.queue[1:]

		if err := s.processItem(item); err != nil {
			if item.optional {
				s.decisions = append(s.decisions, ResolutionDecision{
					Package:   item.name,
					Requested: item.rng,
					Reason:    "optional-failed",
				})
				continue
			}
			return err
		}
	}
	return nil
}

func (s *resolveState) processItem(item workItem) error {
	switch item.protocol {
	case manifest.ProtocolWorkspace:
		return s.processWorkspace(item)
	case manifest.ProtocolFile, manifest.ProtocolLink, manifest.ProtocolPortal:
		return s.processLocal(item)
	default:
		return s.processRegistry(item)
	}
}

func (s *resolveState) processRegistry(item workItem) error {
	if item.depth > maxDepth {
		return apperr.New(apperr.Resolve, "resolver.limit", item.name,
			fmt.Sprintf("resolution depth exceeded %d", maxDepth))
	}

	base := registry.ResolveBaseForPackage(s.e.Effective, s.proj.Root, s.identity, item.name)
	pack, err := s.e.Client.Packument(s.ctx, base, item.name)
	if err != nil {
		return apperr.Wrap(apperr.Resolve, "resolver.packument", item.name, err)
	}

	meta, decision, err := selectVersion(pack, item.name, item.rng, s.pol, &s.hints, pinContext{
		importer:    item.from,
		depName:     item.display,
		kind:        item.kind,
		rangeSpec:   item.rng,
		peerContext: peerContextForEnv(item.envKeys),
	})
	if err != nil {
		return err
	}

	if platformSkipsOptional(item.optional, meta, s.target) {
		decision.Reason = "platform-skipped"
		s.decisions = append(s.decisions, decision)
		return nil
	}

	if err := s.ensurePeersResolved(item.name, meta, item); err != nil {
		return err
	}

	id, key := s.packageKeyForInstance(item, meta.Version, meta)
	decision.Selected = meta.Version
	if item.overrideFrom != "" {
		decision.OverrideFrom = item.overrideFrom
	}
	s.decisions = append(s.decisions, decision)

	tarball := registry.AbsoluteTarballURL(base, item.name, meta.Dist.Tarball)
	if tarball == "" {
		tarball = meta.Dist.Tarball
	}

	s.b.EdgeEx(item.from, item.display, key, item.kind, item.rng, false)
	s.recordProvides(item.from, item.display, key)

	if _, ok := s.seenPkg[key]; ok {
		return nil
	}
	if _, ok := s.resolving[basePackageKey(item.name, meta.Version)]; ok {
		return nil
	}
	if len(s.seenPkg) >= maxPackages {
		return apperr.New(apperr.Resolve, "resolver.limit", item.name,
			fmt.Sprintf("resolution package count exceeded %d", maxPackages))
	}
	s.resolving[basePackageKey(item.name, meta.Version)] = struct{}{}
	defer delete(s.resolving, basePackageKey(item.name, meta.Version))

	s.seenPkg[key] = struct{}{}
	s.b.Package(id, meta.Dist.Integrity, tarball)
	s.pkgEnv[key] = append([]string(nil), item.envKeys...)
	s.pkgFrom[key] = item.from

	if len(meta.PeerDependencies) > 0 {
		peers := make(map[string]string, len(meta.PeerDependencies))
		opt := make(map[string]bool, len(meta.PeerDependencies))
		for k, v := range meta.PeerDependencies {
			peers[k] = v
			opt[k] = peerOptional(meta, k)
		}
		s.pkgPeers[key] = peers
		s.pkgPeerOpt[key] = opt
	}

	nextNamePath := append(append([]string(nil), item.path...), item.name)
	nextEnv := append(append([]string(nil), item.envKeys...), key)
	declarer := s.declarerPathFor(key)
	for _, kv := range sortedDeps(meta.Dependencies) {
		if err := s.enqueue(key, declarer, kv.name, kv.rng, graph.DepProd, item.depth+1, nextNamePath, nextEnv, false); err != nil {
			return err
		}
	}
	for _, kv := range sortedDeps(meta.OptionalDependencies) {
		if err := s.enqueue(key, declarer, kv.name, kv.rng, graph.DepOptional, item.depth+1, nextNamePath, nextEnv, true); err != nil {
			return err
		}
	}
	return nil
}

func parseDependencySpecifier(displayName, spec string) (display, target, rng string, protocol manifest.Protocol, err error) {
	sp, err := manifest.ParseSpecifier(displayName, spec)
	if err != nil {
		return "", "", "", "", err
	}
	return sp.DisplayName, sp.TargetName, sp.Range, sp.Protocol, nil
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
