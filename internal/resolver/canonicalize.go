package resolver

import (
	"fmt"
	"sort"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/graph"
)

// canonicalizePeerInstances merges graph nodes that share the same canonical
// package identity after peer provider contexts are finalized.
func (s *resolveState) canonicalizePeerInstances() error {
	byCanon := map[string][]graph.Package{}
	for _, p := range s.b.Packages() {
		id := p.ID
		id.PeerProviderContext = stripProvisionalProviders(id.PeerProviderContext)
		id.Normalize()
		byCanon[id.Key()] = append(byCanon[id.Key()], p)
	}

	canonKeys := make([]string, 0, len(byCanon))
	for canon := range byCanon {
		canonKeys = append(canonKeys, canon)
	}
	sort.Strings(canonKeys)

	for _, canon := range canonKeys {
		group := byCanon[canon]
		if len(group) <= 1 {
			continue
		}
		ref := group[0]
		refID := ref.ID
		refID.PeerProviderContext = stripProvisionalProviders(refID.PeerProviderContext)
		refID.Normalize()
		for _, p := range group[1:] {
			if p.Integrity != ref.Integrity || p.TarballURL != ref.TarballURL {
				return apperr.New(apperr.Resolve, "resolver.canonicalize", canon,
					fmt.Sprintf("peer-context identity collision for %q", canon))
			}
			oldKey := p.ID.Key()
			if oldKey != canon {
				s.b.RemapPackageKey(oldKey, canon, refID)
				s.remapResolveStateKey(oldKey, canon)
			}
			s.b.RemovePackage(oldKey)
		}
		if ref.ID.Key() != canon {
			s.b.RemapPackageKey(ref.ID.Key(), canon, refID)
			s.remapResolveStateKey(ref.ID.Key(), canon)
		}
	}
	return nil
}

func (s *resolveState) remapResolveStateKey(oldKey, newKey string) {
	if peers, ok := s.pkgPeers[oldKey]; ok {
		delete(s.pkgPeers, oldKey)
		s.pkgPeers[newKey] = peers
	}
	if opt, ok := s.pkgPeerOpt[oldKey]; ok {
		delete(s.pkgPeerOpt, oldKey)
		s.pkgPeerOpt[newKey] = opt
	}
	if env, ok := s.pkgEnv[oldKey]; ok {
		delete(s.pkgEnv, oldKey)
		s.pkgEnv[newKey] = env
	}
	if from, ok := s.pkgFrom[oldKey]; ok {
		delete(s.pkgFrom, oldKey)
		s.pkgFrom[newKey] = from
	}
	delete(s.seenPkg, oldKey)
	s.seenPkg[newKey] = struct{}{}
	for ik, graphKey := range s.pkgInstances {
		if graphKey == oldKey {
			s.pkgInstances[ik] = newKey
		}
	}
	if loc, ok := s.localSources[oldKey]; ok {
		delete(s.localSources, oldKey)
		s.localSources[newKey] = loc
	}
	for ctx, deps := range s.provides {
		for name, dep := range deps {
			if dep.key == oldKey {
				dep.key = newKey
				deps[name] = dep
			}
		}
		s.provides[ctx] = deps
	}
}
