package npm

import (
	"sort"
	"strings"
)

// PackageIndex resolves dependency names to graph package keys.
type PackageIndex struct {
	pathToKey map[string]string
	byName    map[string][]string
}

// BuildIndex indexes packages-map paths to graph keys.
func BuildIndex(packages map[string]PackageEntry) *PackageIndex {
	idx := &PackageIndex{
		pathToKey: make(map[string]string, len(packages)),
		byName:    map[string][]string{},
	}
	if len(packages) > maxIndexKeys {
		return &PackageIndex{pathToKey: map[string]string{}, byName: map[string][]string{}}
	}
	for path, entry := range packages {
		if key, ok := entryPackageKey(path, entry); ok {
			idx.pathToKey[path] = key
			name, _, _ := ParsePackageKey(key)
			idx.byName[name] = append(idx.byName[name], key)
		}
	}
	for name := range idx.byName {
		sort.Strings(idx.byName[name])
	}
	return idx
}

// KeyAtPath returns the graph key at a packages-map path.
func (idx *PackageIndex) KeyAtPath(path string) (string, bool) {
	if idx == nil {
		return "", false
	}
	key, ok := idx.pathToKey[path]
	return key, ok
}

// ResolveDep resolves a dependency from an importer path to a package key.
func (idx *PackageIndex) ResolveDep(packages map[string]PackageEntry, fromPath, depName string) (string, error) {
	path, ok := resolveDepPath(packages, fromPath, depName)
	if !ok {
		return "", errDanglingRef(fromPath, depName)
	}
	entry := packages[path]
	if entry.Link {
		target := strings.TrimPrefix(entry.Resolved, "./")
		target = strings.TrimPrefix(target, "../")
		if key, ok := idx.pathToKey[target]; ok {
			return key, nil
		}
		if key, ok := entryPackageKey(target, packages[target]); ok {
			return key, nil
		}
	}
	key, ok := idx.pathToKey[path]
	if !ok {
		return "", errDanglingRef(fromPath, depName)
	}
	return key, nil
}

func errDanglingRef(fromPath, depName string) error {
	subject := depName
	if fromPath != "" {
		subject = fromPath + " → " + depName
	}
	return errEmptyRef(subject)
}
