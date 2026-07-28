package pnpm

import (
	"bytes"
	"encoding/json"
	"sort"

	"github.com/mewisme/mew/internal/lockfile"
	"go.yaml.in/yaml/v3"
)

// Encode serializes a document to deterministic YAML bytes.
func Encode(doc *Document) ([]byte, error) {
	if doc == nil {
		return nil, nil
	}
	root := map[string]any{}
	if doc.LockfileVersion != "" {
		root["lockfileVersion"] = doc.LockfileVersion
	}
	if len(doc.Settings) > 0 {
		root["settings"] = doc.Settings
	}
	if len(doc.Dependencies) > 0 && IsV6Layout(doc) {
		root["dependencies"] = depsToMap(doc.Dependencies)
	}
	if len(doc.Importers) > 0 {
		root["importers"] = importersToMap(doc.Importers)
	}
	if len(doc.Packages) > 0 {
		root["packages"] = packagesToMap(doc.Packages, doc.Detection)
	}
	if len(doc.Snapshots) > 0 {
		root["snapshots"] = doc.Snapshots
	}
	for k, v := range doc.Extensions {
		var decoded any
		if err := json.Unmarshal(v, &decoded); err == nil {
			root[k] = decoded
		}
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(root); err != nil {
		return nil, err
	}
	_ = enc.Close()
	out := buf.Bytes()
	if len(out) == 0 || out[len(out)-1] != '\n' {
		out = append(out, '\n')
	}
	return out, nil
}

func depsToMap(deps map[string]ImporterDep) map[string]any {
	names := sortedStrings(mapStringKeys(deps))
	out := make(map[string]any, len(names))
	for _, name := range names {
		dep := deps[name]
		out[name] = map[string]any{
			"specifier": dep.Specifier,
			"version":   dep.Version,
		}
	}
	return out
}

func importersToMap(importers map[string]ImporterSection) map[string]any {
	out := make(map[string]any, len(importers))
	for _, id := range sortedStrings(mapImporterKeys(importers)) {
		sec := importers[id]
		m := map[string]any{}
		if len(sec.Dependencies) > 0 {
			m["dependencies"] = depsToMap(sec.Dependencies)
		}
		if len(sec.DevDependencies) > 0 {
			m["devDependencies"] = depsToMap(sec.DevDependencies)
		}
		if len(sec.OptionalDependencies) > 0 {
			m["optionalDependencies"] = depsToMap(sec.OptionalDependencies)
		}
		if len(sec.DependenciesMeta) > 0 {
			m["dependenciesMeta"] = sec.DependenciesMeta
		}
		if sec.PublishDirectory != "" {
			m["publishDirectory"] = sec.PublishDirectory
		}
		for _, k := range sortedExtraKeys(sec.Extra) {
			var decoded any
			if err := json.Unmarshal(sec.Extra[k], &decoded); err == nil {
				m[k] = decoded
			}
		}
		out[id] = m
	}
	return out
}

func sortedExtraKeys(m map[string]json.RawMessage) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func packagesToMap(packages map[string]PackageEntry, det lockfile.Detection) map[string]any {
	out := make(map[string]any, len(packages))
	for _, key := range sortedStrings(mapPackageKeys(packages)) {
		p := packages[key]
		m := map[string]any{}
		if len(p.Resolution) > 0 {
			m["resolution"] = p.Resolution
		}
		if len(p.Engines) > 0 {
			m["engines"] = p.Engines
		}
		if len(p.Dependencies) > 0 {
			depNames := make([]string, 0, len(p.Dependencies))
			for name := range p.Dependencies {
				depNames = append(depNames, name)
			}
			sort.Strings(depNames)
			depMap := make(map[string]string, len(depNames))
			for _, name := range depNames {
				depMap[name] = p.Dependencies[name]
			}
			m["dependencies"] = depMap
		}
		policy := SelectPolicy(det)
		policy.ApplyPackageEncodeFields(p, m)
		extraKeys := sortedStrings(mapAnyKeys(p.Extra))
		for _, fk := range extraKeys {
			m[fk] = p.Extra[fk]
		}
		out[key] = m
	}
	return out
}
