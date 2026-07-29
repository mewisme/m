package pnpm

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
)

// Target is a resolved dependency edge target package key.
type Target struct {
	Key string
}

// PackageIndex resolves dependency references against known package keys.
type PackageIndex interface {
	HasKey(key string) bool
	KeysForName(name string) []string
}

// mapIndex indexes package keys for reference resolution.
type mapIndex struct {
	keys   map[string]struct{}
	byName map[string][]string
}

// NewPackageIndex builds an index from package map keys.
func NewPackageIndex(keys []string) PackageIndex {
	if len(keys) > maxIndexKeys {
		return oversizeIndex{}
	}
	idx := &mapIndex{
		keys:   make(map[string]struct{}, len(keys)),
		byName: map[string][]string{},
	}
	for _, key := range keys {
		idx.keys[key] = struct{}{}
		if name := packageNameFromKey(key); name != "" {
			idx.byName[name] = append(idx.byName[name], key)
		}
	}
	for name := range idx.byName {
		sort.Strings(idx.byName[name])
	}
	return idx
}

func packageNameFromKey(key string) string {
	if isProtocolRef(key) {
		name, _ := protocolNameVersion(key)
		return name
	}
	if base, _, ok := strings.Cut(key, "#"); ok {
		name, _ := splitNameVersionKey(base)
		return name
	}
	id, err := ParsePackageIdentity(key)
	if err != nil {
		name, _ := splitNameVersionKey(key)
		return name
	}
	return id.Name
}

func (idx *mapIndex) HasKey(key string) bool {
	_, ok := idx.keys[key]
	return ok
}

func (idx *mapIndex) KeysForName(name string) []string {
	return idx.byName[name]
}

// oversizeIndex rejects resolution when hostile inputs exceed index caps.
type oversizeIndex struct{}

func (oversizeIndex) HasKey(string) bool { return false }

func (oversizeIndex) KeysForName(string) []string { return nil }

// ResolveDependencyTarget resolves a pnpm dependency reference to a package key.
func ResolveDependencyTarget(depName, resolutionRef string, idx PackageIndex) (Target, error) {
	ref := strings.TrimSpace(resolutionRef)
	if ref == "" {
		return Target{}, apperr.New(apperr.Lockfile, "pnpm.refresolve", depName, "empty dependency reference")
	}
	resolveName := depName
	if actual, _, ok := ParseAliasFromImporterDep(depName, "", ref); ok {
		resolveName = actual
	} else if strings.HasPrefix(ref, "npm:") {
		if actual, resolvedRef, ok := ParseAliasFromImporterDep(depName, ref, ""); ok {
			resolveName = actual
			ref = resolvedRef
		}
	} else if strings.Contains(ref, "@") && !strings.Contains(ref, "(") {
		if id, err := ParsePackageIdentity(ref); err == nil && !id.IsProtocolRef && id.PeerSuffix == "" && id.Name != depName {
			resolveName = id.Name
		}
	}
	if isProtocolRef(ref) {
		if strings.HasPrefix(ref, "workspace:") {
			return resolveWorkspaceTarget(depName, ref, idx)
		}
		return Target{Key: ref}, nil
	}
	if idx != nil && idx.HasKey(ref) {
		return Target{Key: ref}, nil
	}
	if !strings.Contains(ref, "@") {
		candidate := resolveName + "@" + ref
		if idx != nil {
			if gk, err := instanceKeyToGraphKey(candidate); err == nil && idx.HasKey(gk) {
				return Target{Key: gk}, nil
			}
			if idx.HasKey(candidate) {
				return Target{Key: candidate}, nil
			}
		}
		return Target{}, danglingTarget(depName, ref, idx)
	}
	if id, err := ParsePackageIdentity(ref); err == nil && id.Name == resolveName {
		if idx != nil {
			if gk, err := instanceKeyToGraphKey(ref); err == nil && idx.HasKey(gk) {
				return Target{Key: gk}, nil
			}
			if idx.HasKey(ref) {
				return Target{Key: ref}, nil
			}
		}
	}
	if idx != nil {
		if gk, err := instanceKeyToGraphKey(resolveName + "@" + ref); err == nil && idx.HasKey(gk) {
			return Target{Key: gk}, nil
		}
	}
	if idx == nil {
		return Target{}, danglingTarget(depName, ref, nil)
	}
	matches := matchKeysForRef(idx, resolveName, ref)
	switch len(matches) {
	case 0:
		return Target{}, danglingTarget(depName, ref, idx)
	case 1:
		return Target{Key: matches[0]}, nil
	default:
		return Target{}, ambiguousTarget(depName, ref, matches)
	}
}

func matchKeysForRef(idx PackageIndex, depName, ref string) []string {
	keys := idx.KeysForName(depName)
	if len(keys) == 0 {
		return nil
	}
	var matches []string
	for _, key := range keys {
		if key == ref {
			matches = append(matches, key)
			continue
		}
		if gk, err := instanceKeyToGraphKey(depName + "@" + ref); err == nil && gk == key {
			matches = append(matches, key)
			continue
		}
		if gk, err := instanceKeyToGraphKey(ref); err == nil && gk == key {
			matches = append(matches, key)
			continue
		}
		if enc, err := EncodeDependencyRef(depName, key); err == nil && enc == ref {
			matches = append(matches, key)
		}
	}
	return matches
}

func danglingTarget(depName, ref string, idx PackageIndex) error {
	msg := fmt.Sprintf("dangling dependency reference %q", ref)
	if idx != nil {
		if candidates := idx.KeysForName(depName); len(candidates) > 0 {
			msg += fmt.Sprintf("; candidates: %v", candidates)
		}
	}
	return apperr.New(apperr.Lockfile, "pnpm.refresolve", depName, msg)
}

func ambiguousTarget(depName, ref string, matches []string) error {
	return apperr.New(apperr.Lockfile, "pnpm.refresolve", depName,
		fmt.Sprintf("ambiguous dependency reference %q matches %v", ref, matches))
}

var protocolPrefixes = []string{"link:", "workspace:", "file:", "patch:", "git+", "git://", "http://", "https://", "npm:"}

func isProtocolRef(ref string) bool {
	for _, p := range protocolPrefixes {
		if strings.HasPrefix(ref, p) {
			return true
		}
	}
	return false
}

func isLocalProtocolRef(ref string) bool {
	return strings.HasPrefix(ref, "link:") ||
		strings.HasPrefix(ref, "workspace:") ||
		strings.HasPrefix(ref, "file:")
}

func resolveWorkspaceTarget(depName, ref string, idx PackageIndex) (Target, error) {
	_ = ref
	if idx != nil {
		for _, key := range idx.KeysForName(depName) {
			if strings.HasPrefix(key, "link:") || strings.HasPrefix(key, "workspace:") {
				return Target{Key: key}, nil
			}
		}
	}
	linkRef := "link:" + depName
	if idx != nil && idx.HasKey(linkRef) {
		return Target{Key: linkRef}, nil
	}
	return Target{}, apperr.New(apperr.Lockfile, "pnpm.refresolve", depName,
		fmt.Sprintf("dangling workspace reference %q", ref))
}
