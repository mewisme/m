package npm

import (
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/graph"
)

// PackageKey returns the canonical graph key for name@version.
func PackageKey(name, version string) string {
	return name + "@" + version
}

// ParsePackageKey splits name@version[#peer...] into components.
func ParsePackageKey(key string) (name, version string, err error) {
	if key == "" {
		return "", "", apperr.New(apperr.Lockfile, "npm.identity", key, "empty package key")
	}
	base := key
	if i := strings.IndexByte(key, '#'); i >= 0 {
		base = key[:i]
	}
	at := strings.LastIndexByte(base, '@')
	if at <= 0 || at == len(base)-1 {
		return "", "", apperr.New(apperr.Lockfile, "npm.identity", key, "malformed package key")
	}
	return base[:at], base[at+1:], nil
}

// nodeModulesPath returns the node_modules child path for a dependency name.
func nodeModulesPath(parentPath, depName string) string {
	seg := "node_modules/" + depName
	if parentPath == "" {
		return seg
	}
	return parentPath + "/" + seg
}

// resolveDepPath finds the packages-map path for a dependency from an importer path.
func resolveDepPath(packages map[string]PackageEntry, fromPath, depName string) (string, bool) {
	cur := fromPath
	for {
		candidate := nodeModulesPath(cur, depName)
		if _, ok := packages[candidate]; ok {
			return candidate, true
		}
		if cur == "" {
			break
		}
		idx := strings.LastIndex(cur, "/node_modules/")
		if idx < 0 {
			cur = ""
		} else {
			cur = cur[:idx]
		}
	}
	return "", false
}

// entryPackageKey returns the graph package key for a lock entry, or empty when not a package node.
func entryPackageKey(path string, entry PackageEntry) (string, bool) {
	if entry.Link {
		return "", false
	}
	name := entry.Name
	if name == "" && path != "" {
		if i := strings.LastIndex(path, "node_modules/"); i >= 0 {
			name = path[i+len("node_modules/"):]
		} else if idx := strings.LastIndex(path, "/"); idx >= 0 {
			name = path[idx+1:]
		} else {
			name = path
		}
	}
	if name == "" || entry.Version == "" {
		return "", false
	}
	return PackageKey(name, entry.Version), true
}

// importerIDForPath maps a packages-map path to a graph importer id.
func importerIDForPath(path string) graph.ImporterID {
	if path == "" {
		return graph.RootImporter
	}
	return graph.ImporterID(path)
}

// isWorkspaceImporter reports whether a packages entry is a workspace package root.
func isWorkspaceImporter(path string, entry PackageEntry) bool {
	if path == "" {
		return true
	}
	if entry.Link {
		return false
	}
	return entry.Version != "" && !strings.HasPrefix(path, "node_modules/")
}
