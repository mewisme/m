package resolver

import (
	"sort"
	"strings"

	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/registry"
)

func instanceKey(parentEnv, depName, targetName, version, peerFP string) string {
	return parentEnv + "\x00" + depName + "\x00" + targetName + "\x00" + version + "\x00" + peerFP
}

func (s *resolveState) peerContextFingerprint(from string, envKeys []string, meta *registry.VersionMeta) string {
	if meta == nil || len(meta.PeerDependencies) == 0 {
		return ""
	}
	var parts []string
	for _, peer := range sortedPeerDeps(meta.PeerDependencies) {
		if peerOptional(meta, peer.name) {
			continue
		}
		prov, st := s.lookupPeerProvider(peer.name, peer.rng, from, envKeys)
		if st == peerSatisfied {
			parts = append(parts, prov.Key)
		}
	}
	sort.Strings(parts)
	return strings.Join(parts, ",")
}

func (s *resolveState) packageKeyForInstance(item workItem, version string, meta *registry.VersionMeta) (graph.PackageID, string) {
	peerFP := s.peerContextFingerprint(item.from, item.envKeys, meta)
	ik := instanceKey(item.from, item.display, item.name, version, peerFP)
	if key, ok := s.pkgInstances[ik]; ok {
		return parsePackageKey(key), key
	}
	needsInstance := peerFP != "" || (meta != nil && len(meta.PeerDependencies) > 0)
	if !needsInstance {
		id := graph.PackageID{Name: item.name, Version: version}
		id.Normalize()
		key := id.Key()
		s.pkgInstances[ik] = key
		return id, key
	}
	id := graph.PackageID{Name: item.name, Version: version}
	id.PeerProviderContext = graph.PeerProviderContext{graph.ProvisionalPeerProvider(ik)}
	id.Normalize()
	key := id.Key()
	s.pkgInstances[ik] = key
	return id, key
}

func basePackageKey(name, version string) string {
	return graph.PackageID{Name: name, Version: version}.Key()
}

func stripProvisionalProviders(ppc graph.PeerProviderContext) graph.PeerProviderContext {
	if len(ppc) == 0 {
		return nil
	}
	out := make(graph.PeerProviderContext, 0, len(ppc))
	for _, p := range ppc {
		if graph.IsProvisionalPeerKey(p.Key) {
			continue
		}
		out = append(out, p)
	}
	return out
}
