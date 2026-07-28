package resolver

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/policy"
	"github.com/mewisme/mew/internal/registry"
	"github.com/mewisme/mew/internal/semver"
)

// PeerConflict describes an unsatisfied peer dependency.
type PeerConflict struct {
	Package           string           `json:"package"`
	Peer              string           `json:"peer"`
	Range             string           `json:"range"`
	Importer          string           `json:"importer,omitempty"`
	SearchPath        []string         `json:"searchPath,omitempty"`
	SearchSteps       []PeerSearchStep `json:"searchSteps,omitempty"`
	Optional          bool             `json:"optional,omitempty"`
	AutoInstallPolicy bool             `json:"autoInstallPolicy,omitempty"`
	StrictPeers       bool             `json:"strictPeers,omitempty"`
	Incompatible      bool             `json:"incompatible,omitempty"`
	FoundVersion      string           `json:"foundVersion,omitempty"`
}

type peerConflictErr struct {
	PeerConflict
}

func (e *peerConflictErr) Error() string {
	return e.PeerConflict.Error()
}

func (c PeerConflict) Error() string {
	if c.Incompatible {
		return fmt.Sprintf("incompatible peer %s@%q (found %s) required by %s", c.Peer, c.Range, c.FoundVersion, c.Package)
	}
	return fmt.Sprintf("missing peer %s@%q required by %s", c.Peer, c.Range, c.Package)
}

func peerConflictError(pkg, peer, rng, importer string, searchPath []string, steps []PeerSearchStep, optional, autoInstall, strict bool) error {
	conflict := PeerConflict{
		Package:           pkg,
		Peer:              peer,
		Range:             rng,
		Importer:          importer,
		SearchPath:        append([]string(nil), searchPath...),
		SearchSteps:       append([]PeerSearchStep(nil), steps...),
		Optional:          optional,
		AutoInstallPolicy: autoInstall,
		StrictPeers:       strict,
	}
	return apperr.Wrap(apperr.Resolve, "resolver.peer", pkg, &peerConflictErr{conflict})
}

func (s *resolveState) peerConflict(pkg, peer, rng, importer string, envKeys []string, optional, autoInstall, strict bool) error {
	searchPath := s.peerSearchPath(importer, envKeys)
	steps := s.peerSearchSteps(peer, rng, importer, envKeys)
	return peerConflictError(pkg, peer, rng, importer, searchPath, steps, optional, autoInstall, strict)
}

func (s *resolveState) peerSearchSteps(peerName, rng, from string, envKeys []string) []PeerSearchStep {
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
	steps := make([]PeerSearchStep, 0, len(contexts))
	for _, ctx := range contexts {
		step := PeerSearchStep{Environment: ctx}
		deps, ok := s.provides[ctx]
		if !ok {
			step.Candidates = []string{"(no providers in context)"}
			steps = append(steps, step)
			continue
		}
		dep, ok := deps[peerName]
		if !ok {
			step.Candidates = []string{"(peer not provided)"}
			steps = append(steps, step)
			continue
		}
		ok, err := semver.Satisfies(dep.version, rng)
		if err != nil {
			step.Rejected = append(step.Rejected, CandidateRejection{Version: dep.version, Reason: "invalid version"})
		} else if !ok {
			step.Rejected = append(step.Rejected, CandidateRejection{Version: dep.version, Reason: "range mismatch"})
		} else {
			step.Candidates = []string{dep.version}
		}
		steps = append(steps, step)
	}
	return steps
}

func peerIncompatibleError(pkg, peer, rng, foundVersion, importer string, searchPath []string, steps []PeerSearchStep) error {
	conflict := PeerConflict{
		Package:      pkg,
		Peer:         peer,
		Range:        rng,
		Importer:     importer,
		SearchPath:   append([]string(nil), searchPath...),
		SearchSteps:  append([]PeerSearchStep(nil), steps...),
		Incompatible: true,
		FoundVersion: foundVersion,
		StrictPeers:  true,
	}
	return apperr.Wrap(apperr.Resolve, "resolver.peer", pkg, &peerConflictErr{conflict})
}

func (s *resolveState) peerIncompatible(pkg, peer, rng, foundVersion, importer string, envKeys []string) error {
	searchPath := s.peerSearchPath(importer, envKeys)
	steps := s.peerSearchSteps(peer, rng, importer, envKeys)
	return peerIncompatibleError(pkg, peer, rng, foundVersion, importer, searchPath, steps)
}

type peerLookupStatus int

const (
	peerAbsent peerLookupStatus = iota
	peerSatisfied
	peerIncompatibleNearest
)

func (s *resolveState) peerSearchPath(from string, envKeys []string) []string {
	return s.peerSearchContexts(from, envKeys)
}

func (s *resolveState) peerSearchContexts(from string, envKeys []string) []string {
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

func (s *resolveState) lookupPeerProvider(peerName, rng, from string, envKeys []string) (graph.PeerProvider, peerLookupStatus) {
	for _, ctx := range s.peerSearchContexts(from, envKeys) {
		deps, ok := s.provides[ctx]
		if !ok {
			continue
		}
		dep, ok := deps[peerName]
		if !ok {
			continue
		}
		ok, err := semver.Satisfies(dep.version, rng)
		if err != nil {
			continue
		}
		if ok {
			return graph.PeerProvider{Name: peerName, Version: dep.version, Key: dep.key}, peerSatisfied
		}
		if s.pol.StrictPeerDependencies {
			return graph.PeerProvider{Name: peerName, Version: dep.version, Key: dep.key}, peerIncompatibleNearest
		}
	}
	return graph.PeerProvider{}, peerAbsent
}

func (s *resolveState) findPeerProvider(peerName, rng, from string, envKeys []string) (graph.PeerProvider, bool) {
	prov, st := s.lookupPeerProvider(peerName, rng, from, envKeys)
	return prov, st == peerSatisfied
}

func (s *resolveState) ensurePeersResolved(pkgName string, meta *registry.VersionMeta, item workItem) error {
	if meta == nil || len(meta.PeerDependencies) == 0 {
		return nil
	}
	for _, peer := range sortedPeerDeps(meta.PeerDependencies) {
		if peerOptional(meta, peer.name) {
			continue
		}
		prov, st := s.lookupPeerProvider(peer.name, peer.rng, item.from, item.envKeys)
		switch st {
		case peerSatisfied:
			continue
		case peerIncompatibleNearest:
			return s.peerIncompatible(pkgName, peer.name, peer.rng, prov.Version, item.from, item.envKeys)
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
			return s.peerConflict(pkgName, peer.name, peer.rng, item.from, item.envKeys, false, s.pol.AutoInstallPeers, true)
		}
	}
	return nil
}

func (s *resolveState) finalizePeerProviderContexts() {
	keys := make([]string, 0, len(s.seenPkg))
	for k := range s.seenPkg {
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
		base := parsePackageKey(key)
		base.PeerProviderContext = stripProvisionalProviders(base.PeerProviderContext)
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
		for ik, graphKey := range s.pkgInstances {
			if graphKey == key {
				s.pkgInstances[ik] = newKey
			}
		}
		if loc, ok := s.localSources[key]; ok {
			delete(s.localSources, key)
			s.localSources[newKey] = loc
		}
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
			prov, st := s.lookupPeerProvider(peerName, peerRange, from, env)
			switch st {
			case peerSatisfied:
				continue
			case peerIncompatibleNearest:
				if pol.AutoInstallPeers {
					continue
				}
				if pol.StrictPeerDependencies {
					return s.peerIncompatible(id.Name, peerName, peerRange, prov.Version, from, env)
				}
			}
			if pol.AutoInstallPeers {
				continue
			}
			if pol.StrictPeerDependencies {
				return s.peerConflict(id.Name, peerName, peerRange, from, env, false, pol.AutoInstallPeers, true)
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
