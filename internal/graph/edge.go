package graph

import "strings"

const provisionalPeerPrefix = "provisional:"

// TargetNameFromKey extracts the package name from a graph package key (name@version[#peer...]).
func TargetNameFromKey(key string) string {
	to := key
	if i := strings.IndexByte(to, '#'); i >= 0 {
		to = to[:i]
	}
	if i := strings.IndexByte(to, '~'); i >= 0 {
		to = to[:i]
	}
	if strings.HasPrefix(to, "@") {
		slash := strings.IndexByte(to, '/')
		if slash < 0 {
			return to
		}
		at := strings.LastIndexByte(to[slash:], '@')
		if at < 0 {
			return to
		}
		return to[:slash+at]
	}
	at := strings.LastIndexByte(to, '@')
	if at < 0 {
		return to
	}
	return to[:at]
}

// NormalizeEdge fills Edge.Name from the target package name when empty.
func NormalizeEdge(e *Edge) {
	if e == nil || e.Name != "" {
		return
	}
	e.Name = TargetNameFromKey(e.To)
}

// Migrate upgrades an in-memory graph document to the current schema.
func (g *Graph) Migrate() {
	if g == nil || g.SchemaVersion >= SchemaVersion {
		return
	}
	for i := range g.Edges {
		NormalizeEdge(&g.Edges[i])
	}
	g.SchemaVersion = SchemaVersion
}

// IsProvisionalPeerKey reports whether a provider key is a provisional instance marker.
func IsProvisionalPeerKey(key string) bool {
	return strings.HasPrefix(key, provisionalPeerPrefix)
}

// ProvisionalPeerProvider returns a synthetic peer provider for a resolution instance.
func ProvisionalPeerProvider(instanceKey string) PeerProvider {
	return PeerProvider{Key: provisionalPeerPrefix + instanceKey}
}
