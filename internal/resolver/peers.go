package resolver

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/policy"
	"github.com/mewisme/m/internal/registry"
	"github.com/mewisme/m/internal/semver"
)

// PeerConflict describes an unsatisfied peer dependency.
type PeerConflict struct {
	Package           string   `json:"package"`
	Peer              string   `json:"peer"`
	Range             string   `json:"range"`
	Importer          string   `json:"importer,omitempty"`
	SearchPath        []string `json:"searchPath,omitempty"`
	Optional          bool     `json:"optional,omitempty"`
	AutoInstallPolicy bool     `json:"autoInstallPolicy,omitempty"`
}

type peerConflictErr struct {
	PeerConflict
}

func (e *peerConflictErr) Error() string {
	return e.PeerConflict.Error()
}

func (c PeerConflict) Error() string {
	return fmt.Sprintf("missing peer %s@%q required by %s", c.Peer, c.Range, c.Package)
}

func peerConflictError(pkg, peer, rng, importer string, searchPath []string, optional, autoInstall bool) error {
	conflict := PeerConflict{
		Package:           pkg,
		Peer:              peer,
		Range:             rng,
		Importer:          importer,
		SearchPath:        append([]string(nil), searchPath...),
		Optional:          optional,
		AutoInstallPolicy: autoInstall,
	}
	return apperr.Wrap(apperr.Resolve, "resolver.peer", pkg, &peerConflictErr{conflict})
}

func (s *resolveState) peerSearchPath(from string, envKeys []string) []string {
	path := make([]string, 0, 2+len(envKeys))
	path = append(path, from)
	for i := len(envKeys) - 1; i >= 0; i-- {
		path = append(path, envKeys[i])
	}
	if from != string(graph.RootImporter) && s.wsMemberPaths != nil {
		if _, ok := s.wsMemberPaths[from]; ok {
			path = append(path, string(graph.RootImporter))
		}
	}
	return path
}

type providedDep struct {
	key     string
	version string
}

func peerOptional(meta *registry.VersionMeta, peerName string) bool {
	if meta == nil || meta.PeerDependenciesMeta == nil {
		return false
	}
	entry, ok := meta.PeerDependenciesMeta[peerName]
	return ok && entry.Optional
}

func (s *resolveState) recordProvides(from, depName, key string) {
	if from == "" || depName == "" || key == "" {
		return
	}
	if s.provides == nil {
		s.provides = map[string]map[string]providedDep{}
	}
	if s.provides[from] == nil {
		s.provides[from] = map[string]providedDep{}
	}
	id := parsePackageKey(key)
	s.provides[from][depName] = providedDep{key: key, version: id.Version}
}

func (s *resolveState) findPeerProvider(peerName, rng, from string, envKeys []string) (graph.PeerProvider, bool) {
	contexts := make([]string, 0, 2+len(envKeys))
	contexts = append(contexts, from)
	for i := len(envKeys) - 1; i >= 0; i-- {
		contexts = append(contexts, envKeys[i])
	}
	if from != string(graph.RootImporter) && s.wsMemberPaths != nil {
		if _, ok := s.wsMemberPaths[from]; ok {
			contexts = append(contexts, string(graph.RootImporter))
		}
	}
	var best graph.PeerProvider
	found := false
	for _, ctx := range contexts {
		deps, ok := s.provides[ctx]
		if !ok {
			continue
		}
		dep, ok := deps[peerName]
		if !ok {
			continue
		}
		ok, err := semver.Satisfies(dep.version, rng)
		if err != nil || !ok {
			continue
		}
		candidate := graph.PeerProvider{Name: peerName, Version: dep.version, Key: dep.key}
		if !found || versionGT(candidate.Version, best.Version) {
			best = candidate
			found = true
		}
	}
	return best, found
}

func (s *resolveState) ensurePeersResolved(pkgName string, meta *registry.VersionMeta, item workItem) error {
	if meta == nil || len(meta.PeerDependencies) == 0 {
		return nil
	}
	for _, peer := range sortedPeerDeps(meta.PeerDependencies) {
		if peerOptional(meta, peer.name) {
			continue
		}
		if _, ok := s.findPeerProvider(peer.name, peer.rng, item.from, item.envKeys); ok {
			continue
		}
		if s.peerPending(peer.name, item) {
			continue
		}
		if s.pol.AutoInstallPeers {
			declarer := item.declarerPath
			if declarer == "" {
				declarer = s.declarerPathFor(item.from)
			}
			if err := s.enqueue(item.from, declarer, peer.name, peer.rng, graph.DepProd, item.depth, item.path, item.envKeys, false); err != nil {
				return err
			}
			continue
		}
		if s.pol.StrictPeerDependencies {
			return peerConflictError(pkgName, peer.name, peer.rng, item.from, s.peerSearchPath(item.from, item.envKeys), false, s.pol.AutoInstallPeers)
		}
	}
	return nil
}

func (s *resolveState) finalizePeerProviderContexts() {
	keys := make([]string, 0, len(s.pkgPeers))
	for k := range s.pkgPeers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		peerRanges := s.pkgPeers[key]
		opt := s.pkgPeerOpt[key]
		from := s.pkgFrom[key]
		env := s.pkgEnv[key]
		ppc := make(graph.PeerProviderContext, 0, len(peerRanges))
		for peerName, rng := range peerRanges {
			if opt != nil && opt[peerName] {
				if prov, ok := s.findPeerProvider(peerName, rng, from, env); ok {
					ppc = append(ppc, prov)
				}
				continue
			}
			if prov, ok := s.findPeerProvider(peerName, rng, from, env); ok {
				ppc = append(ppc, prov)
			}
		}
		ppc.Sort()
		if len(ppc) == 0 {
			continue
		}
		base := parsePackageKey(key)
		id := graph.PackageID{Name: base.Name, Version: base.Version, PeerProviderContext: ppc}
		id.Normalize()
		newKey := id.Key()
		if newKey == key {
			continue
		}
		s.b.RemapPackageKey(key, newKey, id)
		if peers, ok := s.pkgPeers[key]; ok {
			delete(s.pkgPeers, key)
			s.pkgPeers[newKey] = peers
		}
		if opt, ok := s.pkgPeerOpt[key]; ok {
			delete(s.pkgPeerOpt, key)
			s.pkgPeerOpt[newKey] = opt
		}
		if env, ok := s.pkgEnv[key]; ok {
			delete(s.pkgEnv, key)
			s.pkgEnv[newKey] = env
		}
		if from, ok := s.pkgFrom[key]; ok {
			delete(s.pkgFrom, key)
			s.pkgFrom[newKey] = from
		}
		delete(s.seenPkg, key)
		s.seenPkg[newKey] = struct{}{}
		for ctx, deps := range s.provides {
			for name, dep := range deps {
				if dep.key == key {
					dep.key = newKey
					deps[name] = dep
				}
			}
			s.provides[ctx] = deps
		}
		for i := range s.decisions {
			if s.decisions[i].Package == base.Name && s.decisions[i].Selected == base.Version && len(s.decisions[i].PeerProviders) == 0 {
				s.decisions[i].PeerProviders = ppc
			}
		}
	}
}

func (s *resolveState) peerPending(peerName string, item workItem) bool {
	for _, q := range s.queue {
		if q.name == peerName && q.from == item.from {
			return true
		}
	}
	for key := range s.queuedEdge {
		parts := strings.Split(key, "\x00")
		if len(parts) >= 4 && parts[0] == item.from && parts[2] == peerName {
			return true
		}
	}
	return false
}

func (s *resolveState) validatePeers(pol *policy.Policy) error {
	for key, peers := range s.pkgPeers {
		id := parsePackageKey(key)
		opt := s.pkgPeerOpt[key]
		env := s.pkgEnv[key]
		from := s.pkgFrom[key]
		for peerName, peerRange := range peers {
			if opt != nil && opt[peerName] {
				continue
			}
			if _, ok := s.findPeerProvider(peerName, peerRange, from, env); ok {
				continue
			}
			if pol.AutoInstallPeers {
				continue
			}
			if pol.StrictPeerDependencies {
				return peerConflictError(id.Name, peerName, peerRange, from, s.peerSearchPath(from, env), false, pol.AutoInstallPeers)
			}
		}
	}
	return nil
}

type peerDep struct {
	name, rng string
}

func sortedPeerDeps(m map[string]string) []peerDep {
	if len(m) == 0 {
		return nil
	}
	out := make([]peerDep, 0, len(m))
	for k, v := range m {
		out = append(out, peerDep{name: k, rng: v})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

func parsePackageKey(key string) graph.PackageID {
	var ppc graph.PeerProviderContext
	base := key
	if i := strings.IndexByte(key, '#'); i >= 0 {
		base = key[:i]
		for _, providerKey := range splitTopLevelProviderKeys(key[i+1:]) {
			providerKey = strings.TrimSpace(providerKey)
			if providerKey == "" {
				continue
			}
			pid := parsePackageKey(providerKey)
			ppc = append(ppc, graph.PeerProvider{
				Name:    pid.Name,
				Version: pid.Version,
				Key:     providerKey,
			})
		}
		ppc.Sort()
	}
	name, ver := splitNameVersion(base)
	return graph.PackageID{Name: name, Version: ver, PeerProviderContext: ppc}
}

func splitTopLevelProviderKeys(s string) []string {
	if s == "" {
		return nil
	}
	var keys []string
	start := 0
	inPeerSuffix := false
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '#':
			inPeerSuffix = true
		case ',':
			if !inPeerSuffix {
				keys = append(keys, s[start:i])
				start = i + 1
			}
		}
	}
	keys = append(keys, s[start:])
	return keys
}

func splitNameVersion(base string) (name, version string) {
	if strings.HasPrefix(base, "@") {
		slash := strings.IndexByte(base, '/')
		if slash < 0 {
			return base, ""
		}
		at := strings.LastIndexByte(base[slash:], '@')
		if at < 0 {
			return base, ""
		}
		return base[:slash+at], base[slash+at+1:]
	}
	at := strings.LastIndexByte(base, '@')
	if at < 0 {
		return base, ""
	}
	return base[:at], base[at+1:]
}

func versionGT(a, b string) bool {
	cmp, err := semver.Compare(a, b)
	if err != nil {
		return a > b
	}
	return cmp > 0
}
