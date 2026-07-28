package pnpm

import (
	"strings"
)

// PackageIdentity is a parsed pnpm packages/snapshots map key.
type PackageIdentity struct {
	Name          string
	BaseVersion   string
	PeerSuffix    string
	CanonicalKey  string
	PatchVariant  string
	IsProtocolRef bool
	IsLocalRef    bool
}

// ParsePackageIdentity parses name@version keys, including scoped names and peer suffixes.
func ParsePackageIdentity(key string) (PackageIdentity, error) {
	if err := validatePackageKey(key); err != nil {
		return PackageIdentity{}, err
	}
	id := PackageIdentity{CanonicalKey: key}
	if isProtocolRef(key) {
		id.IsProtocolRef = true
		id.IsLocalRef = isLocalProtocolRef(key)
		id.Name, id.BaseVersion = protocolNameVersion(key)
		return id, nil
	}
	name, ver := splitNameVersionKey(key)
	id.Name = name
	id.BaseVersion, id.PeerSuffix = splitPeerSuffix(ver)
	if patch := patchVariantFromVersion(id.BaseVersion); patch != "" {
		id.PatchVariant = patch
	}
	return id, nil
}

// KeyToNameVersion splits a package key into name and full version (peer suffix included).
func KeyToNameVersion(key string) (string, string) {
	id, err := ParsePackageIdentity(key)
	if err != nil {
		return keyToNameVersionLegacy(key)
	}
	if id.IsProtocolRef {
		return id.Name, id.BaseVersion
	}
	ver := id.BaseVersion + id.PeerSuffix
	return id.Name, ver
}

func splitNameVersionKey(key string) (name, version string) {
	if strings.HasPrefix(key, "@") {
		slash := strings.IndexByte(key, '/')
		if slash < 0 {
			return key, ""
		}
		rest := key[slash+1:]
		at := strings.IndexByte(rest, '@')
		if at < 0 {
			return key, ""
		}
		return key[:slash+1+at], rest[at+1:]
	}
	at := strings.IndexByte(key, '@')
	if at < 0 {
		return key, ""
	}
	return key[:at], key[at+1:]
}

func splitPeerSuffix(version string) (base, peer string) {
	for i := 0; i < len(version); i++ {
		if version[i] == '(' {
			return version[:i], version[i:]
		}
	}
	return version, ""
}

func patchVariantFromVersion(version string) string {
	if i := strings.IndexByte(version, '('); i >= 0 {
		version = version[:i]
	}
	if i := strings.IndexByte(version, '_'); i >= 0 {
		return version[i+1:]
	}
	return ""
}

func protocolNameVersion(ref string) (name, version string) {
	rest := ref
	for _, p := range protocolPrefixes {
		if strings.HasPrefix(rest, p) {
			rest = strings.TrimPrefix(rest, p)
			break
		}
	}
	if at := strings.LastIndexByte(rest, '@'); at > 0 {
		return rest[:at], rest[at+1:]
	}
	return rest, ""
}

// keyToNameVersionLegacy is the pre-identity parser kept for fuzz stability on garbage input.
func keyToNameVersionLegacy(key string) (string, string) {
	if strings.HasPrefix(key, "@") {
		slash := strings.IndexByte(key, '/')
		if slash < 0 {
			return key, ""
		}
		at := strings.LastIndexByte(key[slash:], '@')
		if at < 0 {
			return key, ""
		}
		return key[:slash+at], key[slash+at+1:]
	}
	at := strings.LastIndexByte(key, '@')
	if at < 0 {
		return key, ""
	}
	return key[:at], key[at+1:]
}
