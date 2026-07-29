package pnpm

import (
	"fmt"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
)

// EncodeDependencyRef returns the pnpm lock dependency version reference for a resolved target key.
func EncodeDependencyRef(depName, targetKey string) (string, error) {
	if targetKey == "" {
		return "", apperr.New(apperr.Lockfile, "pnpm.refencode", depName, "empty target key")
	}
	if isProtocolRef(targetKey) {
		return targetKey, nil
	}
	if strings.Contains(targetKey, "#") {
		return dependencyRefFromGraphKey(depName, targetKey)
	}
	id, err := ParsePackageIdentity(targetKey)
	if err == nil {
		if id.Name == depName {
			return id.BaseVersion + id.PeerSuffix, nil
		}
		// npm alias: version field stores actual package@version.
		return id.Name + "@" + id.BaseVersion + id.PeerSuffix, nil
	}
	prefix := depName + "@"
	if strings.HasPrefix(targetKey, prefix) {
		return strings.TrimPrefix(targetKey, prefix), nil
	}
	return "", apperr.New(apperr.Lockfile, "pnpm.refencode", depName,
		fmt.Sprintf("cannot encode target key %q", targetKey))
}

func dependencyRefFromGraphKey(depName, graphKey string) (string, error) {
	if isProtocolRef(graphKey) {
		return graphKey, nil
	}
	base, peerPart, hasPeer := strings.Cut(graphKey, "#")
	name, ver := splitNameVersionKey(base)
	if name != depName {
		return "", fmt.Errorf("graph key %q name mismatch for %q", graphKey, depName)
	}
	if !hasPeer {
		return ver, nil
	}
	return ver + "(" + peerPart + ")", nil
}
