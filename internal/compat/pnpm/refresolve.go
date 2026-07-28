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
	idx := &mapIndex{
		keys:   make(map[string]struct{}, len(keys)),
		byName: map[string][]string{},
	}
	for _, key := range keys {
		idx.keys[key] = struct{}{}
		id, err := ParsePackageIdentity(key)
		if err != nil {
			continue
		}
		idx.byName[id.Name] = append(idx.byName[id.Name], key)
	}
	for name := range idx.byName {
		sort.Strings(idx.byName[name])
	}
	return idx
}

func (idx *mapIndex) HasKey(key string) bool {
	_, ok := idx.keys[key]
	return ok
}

func (idx *mapIndex) KeysForName(name string) []string {
	return idx.byName[name]
}

// ResolveDependencyTarget resolves a pnpm dependency reference to a package key.
func ResolveDependencyTarget(depName, resolutionRef string, idx PackageIndex) (Target, error) {
	ref := strings.TrimSpace(resolutionRef)
	if ref == "" {
		return Target{}, apperr.New(apperr.Lockfile, "pnpm.refresolve", depName, "empty dependency reference")
	}
	if isProtocolRef(ref) {
		return Target{Key: ref}, nil
	}
	if idx != nil && idx.HasKey(ref) {
		return Target{Key: ref}, nil
	}
	if !strings.Contains(ref, "@") {
		candidate := depName + "@" + ref
		if idx != nil {
			if gk, err := instanceKeyToGraphKey(candidate); err == nil && idx.HasKey(gk) {
				return Target{Key: gk}, nil
			}
			if idx.HasKey(candidate) {
				return Target{Key: candidate}, nil
			}
		}
		return Target{}, danglingTarget(depName, ref)
	}
	if id, err := ParsePackageIdentity(ref); err == nil && id.Name == depName {
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
		if gk, err := instanceKeyToGraphKey(depName + "@" + ref); err == nil && idx.HasKey(gk) {
			return Target{Key: gk}, nil
		}
	}
	if idx == nil {
		return Target{}, danglingTarget(depName, ref)
	}
	matches := matchKeysForRef(idx, depName, ref)
	switch len(matches) {
	case 0:
		return Target{}, danglingTarget(depName, ref)
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
		if key == ref || strings.HasSuffix(key, ref) || strings.HasPrefix(key, depName+"@"+ref) {
			matches = append(matches, key)
		}
	}
	return matches
}

func danglingTarget(depName, ref string) error {
	return apperr.New(apperr.Lockfile, "pnpm.refresolve", depName,
		fmt.Sprintf("dangling dependency reference %q", ref))
}

func ambiguousTarget(depName, ref string, matches []string) error {
	return apperr.New(apperr.Lockfile, "pnpm.refresolve", depName,
		fmt.Sprintf("ambiguous dependency reference %q matches %v", ref, matches))
}

var protocolPrefixes = []string{"link:", "workspace:", "file:", "patch:", "git+", "git://", "http://", "https://"}

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
