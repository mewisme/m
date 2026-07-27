package isolated

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"unicode"

	"github.com/mewisme/m/internal/graph"
)

const storeIDMaxLen = 120

// StoreID returns a deterministic virtual-store directory name for id.
// Format: readable name@version[@shortHash] with Windows-safe characters only.
func StoreID(id graph.PackageID) string {
	prefix := nameVersionPrefix(id.Name, id.Version)
	if len(id.PeerProviderContext) == 0 {
		return truncateStoreID(prefix)
	}
	digest := shortPeerDigest(id.PeerProviderContext)
	combined := prefix + "@" + digest
	if len(combined) <= storeIDMaxLen {
		return combined
	}
	keep := storeIDMaxLen - len(digest) - 1
	if keep < 1 {
		return truncateStoreID(digest)
	}
	return truncateStoreID(prefix[:keep] + "@" + digest)
}

// StoreIDFromKey derives StoreID from a package key string.
func StoreIDFromKey(key string) string {
	id := parsePackageKey(key)
	return StoreID(id)
}

func nameVersionPrefix(name, version string) string {
	n := name
	if strings.HasPrefix(n, "@") {
		if i := strings.Index(n, "/"); i > 0 {
			n = n[:i] + "+" + n[i+1:]
		}
	}
	return sanitizeStoreID(n + "@" + version)
}

func shortPeerDigest(ppc graph.PeerProviderContext) string {
	keys := make([]string, len(ppc))
	for i, p := range ppc {
		if p.Key != "" {
			keys[i] = p.Key
			continue
		}
		keys[i] = p.Name + "@" + p.Version
	}
	sum := sha256.Sum256([]byte(strings.Join(keys, ",")))
	return hex.EncodeToString(sum[:8])
}

func sanitizeStoreID(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if isStoreIDRune(r) {
			b.WriteRune(r)
			continue
		}
		b.WriteByte('_')
	}
	return b.String()
}

func isStoreIDRune(r rune) bool {
	if r > unicode.MaxASCII {
		return false
	}
	switch {
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		return true
	case r == '.', r == '-', r == '_', r == '+', r == '@':
		return true
	default:
		return false
	}
}

func truncateStoreID(s string) string {
	if len(s) <= storeIDMaxLen {
		return s
	}
	return s[:storeIDMaxLen]
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
		pid := graph.PackageID{Name: packageNameFromKey(base), Version: packageVersionFromKey(base), PeerProviderContext: ppc}
		pid.Normalize()
		return pid
	}
	return graph.PackageID{Name: packageNameFromKey(base), Version: packageVersionFromKey(base)}
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

func packageNameFromKey(key string) string {
	s := key
	if i := strings.IndexByte(s, '#'); i >= 0 {
		s = s[:i]
	}
	if i := strings.LastIndexByte(s, '@'); i > 0 {
		return s[:i]
	}
	return s
}

func packageVersionFromKey(key string) string {
	s := key
	if i := strings.IndexByte(s, '#'); i >= 0 {
		s = s[:i]
	}
	if i := strings.LastIndexByte(s, '@'); i > 0 {
		return s[i+1:]
	}
	return ""
}
