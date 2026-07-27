package resolver

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"

	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/manifest"
	"github.com/mewisme/m/internal/policy"
)

// reuseKey identifies a prior resolution edge for incremental pin reuse.
// importer + edge + prior package key + peer-provider context + override hash + policy fingerprint.
type reuseKey struct {
	importer     string
	depName      string
	kind         graph.DepKind
	rangeSpec    string
	priorParent  string
	peerContext  string
	overrideHash string
	policyFP     string
}

func (k reuseKey) String() string {
	var b strings.Builder
	b.WriteString(k.importer)
	b.WriteByte(0)
	b.WriteString(k.depName)
	b.WriteByte(0)
	b.WriteString(string(k.kind))
	b.WriteByte(0)
	b.WriteString(k.rangeSpec)
	b.WriteByte(0)
	b.WriteString(k.priorParent)
	b.WriteByte(0)
	b.WriteString(k.peerContext)
	b.WriteByte(0)
	b.WriteString(k.overrideHash)
	b.WriteByte(0)
	b.WriteString(k.policyFP)
	return b.String()
}

// BuildUpdateClosure returns package names that must be re-resolved for an update run.
func BuildUpdateClosure(targets []string, prior *graph.Graph, m *manifest.Manifest) map[string]struct{} {
	return buildUpdateClosure(targets, prior, m)
}

// buildUpdateClosure returns package names that must be re-resolved (not pinned from prior).
// Seeds are UpdateTargets, or all direct manifest dependencies when targets is empty.
// The closure includes each seed and packages reachable from prior lock edges below them.
func buildUpdateClosure(targets []string, prior *graph.Graph, m *manifest.Manifest) map[string]struct{} {
	closure := map[string]struct{}{}
	if prior == nil || m == nil {
		return closure
	}
	seeds := map[string]struct{}{}
	if len(targets) == 0 {
		for _, d := range m.Dependencies {
			if d.Kind == manifest.DepPeer {
				continue
			}
			seeds[d.Name] = struct{}{}
		}
	} else {
		for _, t := range targets {
			seeds[t] = struct{}{}
		}
	}
	for name := range seeds {
		closure[name] = struct{}{}
	}

	importers := importerIDs(prior)
	adj := make(map[string][]string, len(prior.Edges))
	for _, e := range prior.Edges {
		adj[e.From] = append(adj[e.From], e.To)
	}

	seen := map[string]struct{}{}
	var queue []string
	for _, p := range prior.Packages {
		if _, ok := seeds[p.ID.Name]; !ok {
			continue
		}
		key := p.ID.Key()
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		queue = append(queue, key)
	}
	for _, e := range prior.Edges {
		if _, ok := importers[graph.ImporterID(e.From)]; !ok {
			continue
		}
		id := parsePackageKey(e.To)
		if _, ok := seeds[id.Name]; !ok {
			continue
		}
		if _, ok := seen[e.To]; ok {
			continue
		}
		seen[e.To] = struct{}{}
		queue = append(queue, e.To)
	}

	for len(queue) > 0 {
		key := queue[0]
		queue = queue[1:]
		id := parsePackageKey(key)
		closure[id.Name] = struct{}{}
		for _, to := range adj[key] {
			if _, ok := seen[to]; ok {
				continue
			}
			seen[to] = struct{}{}
			queue = append(queue, to)
		}
	}
	return closure
}

func buildReuseIndex(prior *graph.Graph, overrideHash, policyFP string) map[string]string {
	if prior == nil {
		return nil
	}
	importers := importerIDs(prior)
	pkgByKey := map[string]graph.Package{}
	for _, p := range prior.Packages {
		pkgByKey[p.ID.Key()] = p
	}
	out := make(map[string]string, len(prior.Edges))
	for _, e := range prior.Edges {
		toID := parsePackageKey(e.To)
		peerCtx := ""
		if pkg, ok := pkgByKey[e.To]; ok {
			peerCtx = pkg.ID.PeerProviderContext.String()
		}
		priorParent := e.From
		if _, ok := importers[graph.ImporterID(e.From)]; ok {
			priorParent = e.From
		}
		k := reuseKey{
			importer:     e.From,
			depName:      toID.Name,
			kind:         e.Kind,
			rangeSpec:    e.Range,
			priorParent:  priorParent,
			peerContext:  peerCtx,
			overrideHash: overrideHash,
			policyFP:     policyFP,
		}
		out[k.String()] = e.To
	}
	return out
}

func hashOverrides(m map[string]string) string {
	if len(m) == 0 {
		return ""
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte(0)
		b.WriteString(m[k])
		b.WriteByte(0)
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:8])
}

func policyFingerprint(pol *policy.Policy) string {
	if pol == nil {
		return ""
	}
	data, err := policy.EncodeJSON(pol)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:8])
}

// mergeUnchangedSubgraph copies packages and edges outside the update closure from prior
// so targeted updates keep unrelated subgraph byte-stable.
func mergeUnchangedSubgraph(prior, resolved *graph.Graph, closure map[string]struct{}) (*graph.Graph, error) {
	if prior == nil || resolved == nil || len(closure) == 0 {
		return resolved, nil
	}
	importers := importerIDs(prior)
	preservedKeys := map[string]struct{}{}
	for _, p := range prior.Packages {
		if _, inClosure := closure[p.ID.Name]; inClosure {
			continue
		}
		preservedKeys[p.ID.Key()] = struct{}{}
	}

	b := graph.NewBuilder()
	for _, im := range resolved.Importers {
		b.Importer(im.ID, im.Name)
	}
	for _, p := range prior.Packages {
		if _, ok := preservedKeys[p.ID.Key()]; ok {
			b.Package(p.ID, p.Integrity, p.TarballURL)
		}
	}
	for _, p := range resolved.Packages {
		if _, inClosure := closure[p.ID.Name]; !inClosure {
			continue
		}
		b.Package(p.ID, p.Integrity, p.TarballURL)
	}
	for _, e := range prior.Edges {
		if preservedEdge(e, closure, importers) {
			b.EdgeEx(e.From, e.To, e.Kind, e.Range, e.Optional)
		}
	}
	for _, e := range resolved.Edges {
		if preservedEdge(e, closure, importers) {
			continue
		}
		b.EdgeEx(e.From, e.To, e.Kind, e.Range, e.Optional)
	}
	return b.Build()
}

func preservedEdge(e graph.Edge, closure map[string]struct{}, importers map[graph.ImporterID]struct{}) bool {
	toName := parsePackageKey(e.To).Name
	if _, inClosure := closure[toName]; inClosure {
		return false
	}
	if _, ok := importers[graph.ImporterID(e.From)]; ok {
		return true
	}
	fromName := parsePackageKey(e.From).Name
	_, fromInClosure := closure[fromName]
	return !fromInClosure
}

// ExtractPackageSubgraph returns the package node and incident edges for one resolved name.
func ExtractPackageSubgraph(g *graph.Graph, pkgName string) (*graph.Graph, error) {
	return extractPackageSubgraph(g, pkgName)
}

func extractPackageSubgraph(g *graph.Graph, pkgName string) (*graph.Graph, error) {
	if g == nil {
		return nil, nil
	}
	var targetKey string
	for _, p := range g.Packages {
		if p.ID.Name == pkgName {
			targetKey = p.ID.Key()
			break
		}
	}
	if targetKey == "" {
		return graph.NewBuilder().Build()
	}
	b := graph.NewBuilder()
	for _, im := range g.Importers {
		b.Importer(im.ID, im.Name)
	}
	for _, p := range g.Packages {
		if p.ID.Key() == targetKey {
			b.Package(p.ID, p.Integrity, p.TarballURL)
			break
		}
	}
	for _, e := range g.Edges {
		if e.To == targetKey || e.From == targetKey {
			b.EdgeEx(e.From, e.To, e.Kind, e.Range, e.Optional)
		}
	}
	return b.Build()
}

func importerIDs(g *graph.Graph) map[graph.ImporterID]struct{} {
	out := make(map[graph.ImporterID]struct{}, len(g.Importers))
	for _, im := range g.Importers {
		out[im.ID] = struct{}{}
	}
	return out
}

func rootImporterSpecifiers(g *graph.Graph) map[string]string {
	out := map[string]string{}
	if g == nil {
		return out
	}
	importers := importerIDs(g)
	for _, e := range g.Edges {
		if _, ok := importers[graph.ImporterID(e.From)]; !ok {
			continue
		}
		if e.From != string(graph.RootImporter) {
			continue
		}
		id := parsePackageKey(e.To)
		out[id.Name] = e.Range
	}
	return out
}

func manifestSpecifierMap(m *manifest.Manifest) map[string]string {
	out := map[string]string{}
	if m == nil {
		return out
	}
	for _, d := range m.Dependencies {
		out[d.Name] = d.Range
	}
	return out
}

func mapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func prepareHints(opts ResolveOptions, m *manifest.Manifest) graphHints {
	h := graphHints{g: opts.Hints}
	if opts.Prior != nil {
		if h.g == nil {
			h.g = opts.Prior
		}
		if opts.IncrementalUpdate {
			h.incremental = true
			h.updateClosure = buildUpdateClosure(opts.UpdateTargets, opts.Prior, m)
			h.priorSpecs = rootImporterSpecifiers(opts.Prior)
			h.manifestSpecs = manifestSpecifierMap(m)
			h.priorOverrides = opts.PriorOverrides
			h.currentOverrides = m.Overrides
			ovrHash := hashOverrides(m.Overrides)
			if opts.PriorOverrides != nil {
				ovrHash = hashOverrides(opts.PriorOverrides)
				if !mapsEqual(m.Overrides, opts.PriorOverrides) {
					h.overrideChanged = true
				}
			}
			h.policyFP = policyFingerprint(opts.Policy)
			h.reuseIndex = buildReuseIndex(opts.Prior, ovrHash, h.policyFP)
		}
	}
	return h
}
