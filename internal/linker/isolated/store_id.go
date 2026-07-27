package isolated

import (
	"strings"

	"github.com/mewisme/m/internal/graph"
)

// StoreID returns a deterministic virtual-store directory name for id.
func StoreID(id graph.PackageID) string {
	name := id.Name
	if strings.HasPrefix(name, "@") {
		if i := strings.Index(name, "/"); i > 0 {
			name = name[:i] + "+" + name[i+1:]
		}
	}
	sid := name + "@" + id.Version
	if sfx := id.PeerContext.String(); sfx != "" {
		sid += "(" + strings.ReplaceAll(sfx, ",", "_") + ")"
	}
	return sid
}

// StoreIDFromKey derives StoreID from a package key string.
func StoreIDFromKey(key string) string {
	name := packageNameFromKey(key)
	version := packageVersionFromKey(key)
	peer := peerSuffixFromKey(key)
	id := graph.PackageID{Name: name, Version: version}
	if peer != "" {
		for _, part := range strings.Split(peer, ",") {
			if i := strings.LastIndex(part, "@"); i > 0 {
				id.PeerContext = append(id.PeerContext, graph.PeerRef{
					Name:  part[:i],
					Range: part[i+1:],
				})
			}
		}
		id.Normalize()
	}
	return StoreID(id)
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

func peerSuffixFromKey(key string) string {
	if i := strings.IndexByte(key, '#'); i >= 0 {
		return key[i+1:]
	}
	return ""
}
