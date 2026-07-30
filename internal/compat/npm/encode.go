package npm

import (
	"encoding/json"
	"sort"
)

func encodePayload(doc *Document) map[string]any {
	out := map[string]any{
		"lockfileVersion": doc.LockfileVersion,
	}
	if doc.Name != "" {
		out["name"] = doc.Name
	}
	if doc.Version != "" {
		out["version"] = doc.Version
	}
	if doc.Requires {
		out["requires"] = true
	}
	if len(doc.Packages) > 0 {
		pkgs := map[string]any{}
		for _, path := range sortedPackagePaths(doc.Packages) {
			pkgs[path] = encodePackageEntry(doc.Packages[path])
		}
		out["packages"] = pkgs
	}
	if doc.LockfileVersion == 2 && len(doc.Dependencies) > 0 {
		out["dependencies"] = encodeLegacyDeps(doc.Dependencies)
	}
	for _, k := range sortedExtensionKeys(doc.Extensions) {
		var v any
		_ = json.Unmarshal(doc.Extensions[k], &v)
		out[k] = v
	}
	return out
}

func sortedPackagePaths(m map[string]PackageEntry) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedExtensionKeys(m map[string]json.RawMessage) []string {
	if len(m) == 0 {
		return nil
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func encodePackageEntry(entry PackageEntry) map[string]any {
	out := map[string]any{}
	if entry.Name != "" {
		out["name"] = entry.Name
	}
	if entry.Version != "" {
		out["version"] = entry.Version
	}
	if entry.Resolved != "" {
		out["resolved"] = entry.Resolved
	}
	if entry.Integrity != "" {
		out["integrity"] = entry.Integrity
	}
	if entry.Link {
		out["link"] = true
	}
	if entry.Dev {
		out["dev"] = true
	}
	if entry.DevOptional {
		out["devOptional"] = true
	}
	if entry.Optional {
		out["optional"] = true
	}
	encodeStringMap(out, "dependencies", entry.Dependencies)
	encodeStringMap(out, "devDependencies", entry.DevDependencies)
	encodeStringMap(out, "optionalDependencies", entry.OptionalDependencies)
	encodeStringMap(out, "peerDependencies", entry.PeerDependencies)
	if len(entry.PeerDependenciesMeta) > 0 {
		keys := sortedStringKeysPeerMeta(entry.PeerDependenciesMeta)
		meta := make(map[string]any, len(keys))
		for _, k := range keys {
			m := entry.PeerDependenciesMeta[k]
			entryOut := map[string]any{}
			if m.Optional {
				entryOut["optional"] = true
			}
			meta[k] = entryOut
		}
		out["peerDependenciesMeta"] = meta
	}
	if len(entry.BundledDependencies) > 0 {
		bundled := append([]string(nil), entry.BundledDependencies...)
		sort.Strings(bundled)
		out["bundledDependencies"] = bundled
	}
	if len(entry.Workspaces) > 0 {
		ws := append([]string(nil), entry.Workspaces...)
		sort.Strings(ws)
		out["workspaces"] = ws
	}
	for _, k := range sortedExtensionKeys(entry.Extra) {
		var v any
		_ = json.Unmarshal(entry.Extra[k], &v)
		out[k] = v
	}
	return out
}

func encodeStringMap(out map[string]any, key string, m map[string]string) {
	if len(m) == 0 {
		return
	}
	keys := sortedStringKeys(m)
	encoded := make(map[string]string, len(m))
	for _, k := range keys {
		encoded[k] = m[k]
	}
	out[key] = encoded
}

func sortedStringKeysPeerMeta(m map[string]PeerMeta) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func encodeLegacyDeps(m map[string]LegacyDep) map[string]any {
	out := map[string]any{}
	for _, name := range sortedLegacyDepNames(m) {
		out[name] = encodeLegacyDep(m[name])
	}
	return out
}

func sortedLegacyDepNames(m map[string]LegacyDep) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func encodeLegacyDep(dep LegacyDep) map[string]any {
	out := map[string]any{}
	if dep.Version != "" {
		out["version"] = dep.Version
	}
	if dep.Resolved != "" {
		out["resolved"] = dep.Resolved
	}
	if dep.Integrity != "" {
		out["integrity"] = dep.Integrity
	}
	if dep.Requires {
		out["requires"] = true
	}
	if len(dep.Deps) > 0 {
		out["dependencies"] = encodeLegacyDeps(dep.Deps)
	}
	return out
}
