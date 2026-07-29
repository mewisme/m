package resolver

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/manifest"
	"github.com/mewisme/mew/internal/policy"
)

// reuseKey identifies a prior resolution edge for incremental pin reuse.
// importer + edge name + kind + range + peer context + override hash + policy fingerprint.
type reuseKey struct {
	importer     string
	depName      string
	kind         graph.DepKind
	rangeSpec    string
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
	b.WriteString(k.peerContext)
	b.WriteByte(0)
	b.WriteString(k.overrideHash)
	b.WriteByte(0)
	b.WriteString(k.policyFP)
	return b.String()
}

type pinContext struct {
	importer    string
	depName     string
	kind        graph.DepKind
	rangeSpec   string
	peerContext string
}

// BuildUpdateClosure returns package instance keys that must be re-resolved for an update run.
func BuildUpdateClosure(targets []string, prior *graph.Graph, m *manifest.Manifest) map[string]struct{} {
	return buildUpdateClosure(targets, prior, m)
}

// buildUpdateClosure returns package instance keys that must be re-resolved (not pinned from prior).
// Seeds are UpdateTargets (by exposed edge name), or all direct manifest dependencies when targets is empty.
// The closure includes each seeded package key and descendants reachable via prior graph edges.
func buildUpdateClosure(targets []string, prior *graph.Graph, m *manifest.Manifest) map[string]struct{} {
	closure := map[string]struct{}{}
	if prior == nil || m == nil {
		return closure
	}

	targetSet := map[string]struct{}{}
	if len(targets) == 0 {
		for _, d := range m.Dependencies {
			if d.Kind == manifest.DepPeer {
				continue
			}
			targetSet[d.Name] = struct{}{}
		}
	} else {
		for _, t := range targets {
			targetSet[t] = struct{}{}
		}
	}

	queue := make([]string, 0)
	seen := map[string]struct{}{}
	for _, e := range prior.Edges {
		if e.From != string(graph.RootImporter) {
			continue
		}
		if _, ok := targetSet[e.Name]; !ok {
			if _, ok := seen[e.To]; ok {
				continue
			}
			seen[e.To] = struct{}{}
			queue = append(queue, e.To)
			closure[e.To] = struct{}{}
			continue
		}
		if _, ok := seen[e.To]; ok {
			continue
		}
		seen[e.To] = struct{}{}
		queue = append(queue, e.To)
		closure[e.To] = struct{}{}
	}

	for len(queue) > 0 {
		key := queue[0]
		queue = queue[1:]
		for _, e := range prior.Edges {
			if e.From != key {
				continue
			}
			if _, ok := closure[e.To]; ok {
				continue
			}
			closure[e.To] = struct{}{}
			queue = append(queue, e.To)
		}
	}
	return closure
}

func peerContextForEnv(envKeys []string) string {
	if len(envKeys) == 0 {
		return ""
	}
	parts := make([]string, len(envKeys))
	for i, k := range envKeys {
		id := parsePackageKey(k)
		if s := id.PeerProviderContext.String(); s != "" {
			parts[i] = s
			continue
		}
		parts[i] = k
	}
	return strings.Join(parts, ",")
}

func buildReuseIndex(prior *graph.Graph, overrideHash, policyFP string) map[string]string {
	if prior == nil {
		return nil
	}
	pkgByKey := map[string]graph.Package{}
	for _, p := range prior.Packages {
		pkgByKey[p.ID.Key()] = p
	}
	out := make(map[string]string, len(prior.Edges))
	for _, e := range prior.Edges {
		peerCtx := ""
		if pkg, ok := pkgByKey[e.To]; ok {
			peerCtx = pkg.ID.PeerProviderContext.String()
		}
		k := reuseKey{
			importer:     e.From,
			depName:      e.Name,
			kind:         e.Kind,
			rangeSpec:    e.Range,
			peerContext:  peerCtx,
			overrideHash: overrideHash,
			policyFP:     policyFP,
		}
		out[k.String()] = e.To
	}
	return out
}

// OverridesFingerprint hashes an overrides map for lockfile settings.
func OverridesFingerprint(m map[string]string) string { return hashOverrides(m) }

// PolicyFingerprint hashes resolver policy for lockfile settings.
func PolicyFingerprint(pol *policy.Policy) string { return policyFingerprint(pol) }

// TargetPlatformFingerprint hashes the install target platform for lockfile settings.
func TargetPlatformFingerprint(t Target) string { return targetPlatformFingerprint(t) }

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

// resolverPolicyFingerprintSchema versions the resolver-subset fingerprint encoder.
const resolverPolicyFingerprintSchema = 1

// resolverPolicySubset is the graph-affecting policy surface encoded into lock fingerprints.
type resolverPolicySubset struct {
	SchemaVersion          int           `json:"schemaVersion"`
	StrictPeerDependencies bool          `json:"strictPeerDependencies"`
	AutoInstallPeers       bool          `json:"autoInstallPeers"`
	MinimumReleaseAge      time.Duration `json:"minimumReleaseAge,omitempty"`
	RejectDeprecated       bool          `json:"rejectDeprecated,omitempty"`
	Offline                bool          `json:"offline,omitempty"`
}

func resolverPolicySubsetFrom(pol *policy.Policy) resolverPolicySubset {
	if pol == nil {
		return resolverPolicySubset{SchemaVersion: resolverPolicyFingerprintSchema, StrictPeerDependencies: true}
	}
	return resolverPolicySubset{
		SchemaVersion:          resolverPolicyFingerprintSchema,
		StrictPeerDependencies: pol.StrictPeerDependencies,
		AutoInstallPeers:       pol.AutoInstallPeers,
		MinimumReleaseAge:      pol.MinimumReleaseAge,
		RejectDeprecated:       pol.RejectDeprecated,
		Offline:                pol.Offline,
	}
}

func policyFingerprint(pol *policy.Policy) string {
	sub := resolverPolicySubsetFrom(pol)
	data, err := json.Marshal(sub)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:8])
}

func validPolicyFingerprint(fp string) bool {
	if len(fp) != 16 {
		return false
	}
	for _, c := range fp {
		switch {
		case c >= '0' && c <= '9':
		case c >= 'a' && c <= 'f':
		case c >= 'A' && c <= 'F':
		default:
			return false
		}
	}
	return true
}

func targetPlatformFingerprint(t Target) string {
	sum := sha256.Sum256([]byte(t.OS + "\x00" + t.CPU + "\x00" + t.Libc))
	return hex.EncodeToString(sum[:8])
}

// mergeUnchangedSubgraph copies packages and edges outside the update closure from prior
// so targeted updates keep unrelated subgraph byte-stable.
func mergeUnchangedSubgraph(prior, resolved *graph.Graph, closure map[string]struct{}) (*graph.Graph, error) {
	if prior == nil || resolved == nil || len(closure) == 0 {
		return resolved, nil
	}
	mergeClosure := expandClosureForMerge(prior, resolved, closure)
	importers := importerIDs(prior)

	b := graph.NewBuilder()
	for _, im := range resolved.Importers {
		b.Importer(im.ID, im.Name)
	}
	for _, p := range resolved.Packages {
		if _, in := mergeClosure[p.ID.Key()]; !in {
			continue
		}
		b.Package(p.ID, p.Integrity, p.TarballURL)
	}
	for _, p := range prior.Packages {
		if _, in := mergeClosure[p.ID.Key()]; in {
			continue
		}
		b.Package(p.ID, p.Integrity, p.TarballURL)
	}
	for _, e := range prior.Edges {
		if preservedEdge(e, mergeClosure, importers) {
			b.EdgeEx(e.From, e.Name, e.To, e.Kind, e.Range, e.Optional)
		}
	}
	for _, e := range resolved.Edges {
		if preservedEdge(e, mergeClosure, importers) {
			continue
		}
		b.EdgeEx(e.From, e.Name, e.To, e.Kind, e.Range, e.Optional)
	}
	return b.Build()
}

// expandClosureForMerge maps prior closure keys to resolved package keys for the same
// edge identities so version bumps inside the update subtree merge correctly.
func expandClosureForMerge(prior, resolved *graph.Graph, closure map[string]struct{}) map[string]struct{} {
	out := make(map[string]struct{}, len(closure))
	for k := range closure {
		out[k] = struct{}{}
	}
	importers := importerIDs(prior)
	resolvedImporters := importerIDs(resolved)
	keyMap := buildPriorResolvedKeyMap(prior, resolved, closure, importers)

	queue := make([]string, 0, len(closure))
	for k := range closure {
		queue = append(queue, k)
	}
	seen := make(map[string]struct{}, len(closure))
	for len(queue) > 0 {
		key := queue[0]
		queue = queue[1:]
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		for _, pe := range prior.Edges {
			if pe.From != key && pe.To != key {
				continue
			}
			parentKey := mappedParentKey(pe.From, keyMap, importers)
			if re, ok := matchingResolvedEdge(pe, resolved, parentKey, closure); ok {
				out[re.To] = struct{}{}
				keyMap[pe.To] = re.To
				if _, imp := resolvedImporters[graph.ImporterID(re.From)]; !imp {
					out[re.From] = struct{}{}
					keyMap[pe.From] = re.From
					if _, done := seen[re.From]; !done {
						queue = append(queue, re.From)
					}
				}
				if _, done := seen[re.To]; !done {
					queue = append(queue, re.To)
				}
			}
			if _, in := closure[pe.To]; in {
				out[pe.To] = struct{}{}
				if _, done := seen[pe.To]; !done {
					queue = append(queue, pe.To)
				}
			}
			if _, imp := importers[graph.ImporterID(pe.From)]; imp {
				continue
			}
			if _, in := closure[pe.From]; in {
				out[pe.From] = struct{}{}
				if _, done := seen[pe.From]; !done {
					queue = append(queue, pe.From)
				}
			}
		}
	}
	return out
}

// buildPriorResolvedKeyMap links prior package keys to resolved keys via full parent identity.
func buildPriorResolvedKeyMap(
	prior, resolved *graph.Graph,
	closure map[string]struct{},
	importers map[graph.ImporterID]struct{},
) map[string]string {
	mapping := map[string]string{}
	changed := true
	for changed {
		changed = false
		for _, pe := range prior.Edges {
			touchesClosure := false
			if _, in := closure[pe.To]; in {
				touchesClosure = true
			} else if _, in := closure[pe.From]; in {
				touchesClosure = true
			} else if _, imp := importers[graph.ImporterID(pe.From)]; imp {
				touchesClosure = true
			}
			if !touchesClosure {
				continue
			}
			parentKey := mappedParentKey(pe.From, mapping, importers)
			if re, ok := matchingResolvedEdge(pe, resolved, parentKey, closure); ok {
				if priorTo, exists := mapping[pe.To]; exists && priorTo != re.To {
					continue
				}
				if _, exists := mapping[pe.To]; !exists {
					mapping[pe.To] = re.To
					changed = true
				}
			}
		}
	}
	return mapping
}

func mappedParentKey(from string, keyMap map[string]string, importers map[graph.ImporterID]struct{}) string {
	if mapped, ok := keyMap[from]; ok {
		return mapped
	}
	return edgeParentKey(from)
}

func matchingResolvedEdge(pe graph.Edge, resolved *graph.Graph, parentKey string, updateClosure map[string]struct{}) (graph.Edge, bool) {
	if parentKey == "" {
		parentKey = edgeParentKey(pe.From)
	}
	allowRangeChange := edgeTouchesClosure(pe, updateClosure)
	for _, re := range resolved.Edges {
		if edgeParentKey(re.From) != parentKey {
			continue
		}
		if re.Name != pe.Name || re.Kind != pe.Kind || re.Optional != pe.Optional {
			continue
		}
		if !allowRangeChange && re.Range != pe.Range {
			continue
		}
		return re, true
	}
	return graph.Edge{}, false
}

func edgeTouchesClosure(e graph.Edge, closure map[string]struct{}) bool {
	if len(closure) == 0 {
		return false
	}
	if _, ok := closure[e.From]; ok {
		return true
	}
	_, ok := closure[e.To]
	return ok
}

func edgeParentKey(endpoint string) string {
	return endpoint
}

func preservedEdge(e graph.Edge, closure map[string]struct{}, importers map[graph.ImporterID]struct{}) bool {
	if _, in := closure[e.To]; in {
		return false
	}
	if _, ok := importers[graph.ImporterID(e.From)]; ok {
		return true
	}
	if _, in := closure[e.From]; in {
		return false
	}
	return true
}

// ExtractPackageSubgraph returns the package node and incident edges for one package key.
func ExtractPackageSubgraph(g *graph.Graph, pkgKey string) (*graph.Graph, error) {
	return extractPackageSubgraph(g, pkgKey)
}

func extractPackageSubgraph(g *graph.Graph, pkgKey string) (*graph.Graph, error) {
	if g == nil {
		return nil, nil
	}
	targetKey := pkgKey
	found := false
	for _, p := range g.Packages {
		if p.ID.Key() == pkgKey {
			found = true
			break
		}
	}
	if !found {
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
			b.EdgeEx(e.From, e.Name, e.To, e.Kind, e.Range, e.Optional)
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
		out[e.Name] = e.Range
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

func prepareHints(eff *config.Effective, opts ResolveOptions, m *manifest.Manifest) graphHints {
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
			currentPolicy := PolicyFromEffective(eff)
			h.policyFP = policyFingerprint(currentPolicy)
			if pf := opts.PriorFingerprints; pf == nil {
				h.policyDrift = true
			} else {
				if pf.OverridesFingerprint != "" && pf.OverridesFingerprint != hashOverrides(m.Overrides) {
					h.overrideChanged = true
				}
				if pf.ResolverPolicyFingerprint == "" || !validPolicyFingerprint(pf.ResolverPolicyFingerprint) {
					h.policyDrift = true
				} else if pf.ResolverPolicyFingerprint != h.policyFP {
					h.policyDrift = true
				}
				if pf.TargetPlatformFingerprint != "" && pf.TargetPlatformFingerprint != targetPlatformFingerprint(CurrentTarget()) {
					h.platformDrift = true
				}
			}
			h.reuseIndex = buildReuseIndex(opts.Prior, ovrHash, h.policyFP)
		}
	}
	return h
}
