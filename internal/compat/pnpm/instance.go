package pnpm

import (
	"fmt"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/graph"
)

func buildInstanceSet(doc *Document) []string {
	if len(doc.Snapshots) == 0 {
		seen := make(map[string]struct{}, len(doc.Packages))
		for k := range doc.Packages {
			seen[k] = struct{}{}
		}
		out := make([]string, 0, len(seen))
		for k := range seen {
			out = append(out, k)
		}
		return sortedStrings(out)
	}

	seen := make(map[string]struct{}, len(doc.Snapshots))
	for k := range doc.Snapshots {
		seen[k] = struct{}{}
	}
	for _, snap := range doc.Snapshots {
		collectSnapshotTargetInstances(doc, snap, seen)
	}
	for _, im := range doc.Importers {
		collectImporterDepInstances(doc, im, seen)
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	return sortedStrings(out)
}

func collectSnapshotTargetInstances(doc *Document, snap map[string]any, seen map[string]struct{}) {
	for _, field := range []string{"dependencies", "optionalDependencies", "peerDependencies"} {
		raw, ok := snap[field]
		if !ok {
			continue
		}
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		for depName, v := range m {
			ref, ok := v.(string)
			if !ok {
				continue
			}
			maybeAddPackageOnlyInstance(doc, depName, ref, seen)
		}
	}
}

func collectImporterDepInstances(doc *Document, im ImporterSection, seen map[string]struct{}) {
	for name, dep := range im.Dependencies {
		maybeAddPackageOnlyInstance(doc, name, dep.Version, seen)
	}
	for name, dep := range im.DevDependencies {
		maybeAddPackageOnlyInstance(doc, name, dep.Version, seen)
	}
	for name, dep := range im.OptionalDependencies {
		maybeAddPackageOnlyInstance(doc, name, dep.Version, seen)
	}
}

func maybeAddPackageOnlyInstance(doc *Document, depName, ref string, seen map[string]struct{}) {
	if isProtocolRef(ref) {
		return
	}
	baseKey := refToBasePackageKey(depName, ref)
	if baseKey == "" {
		return
	}
	if _, ok := doc.Packages[baseKey]; !ok {
		return
	}
	if _, ok := seen[baseKey]; ok {
		return
	}
	seen[baseKey] = struct{}{}
}

func refToBasePackageKey(depName, ref string) string {
	resolveName := depName
	if actual, _, ok := ParseAliasFromImporterDep(depName, "", ref); ok {
		resolveName = actual
	}
	if strings.HasPrefix(ref, "npm:") {
		if actual, resolutionRef, ok := ParseAliasFromImporterDep(depName, ref, ""); ok {
			resolveName = actual
			ref = resolutionRef
		}
	}
	if id, err := ParsePackageIdentity(ref); err == nil && !id.IsProtocolRef {
		return id.Name + "@" + id.BaseVersion
	}
	if !strings.Contains(ref, "@") {
		return resolveName + "@" + ref
	}
	name, ver := splitNameVersionKey(ref)
	if name != "" && ver != "" {
		return name + "@" + ver
	}
	return ""
}

func basePackageKeyFromInstance(instanceKey string) string {
	id, err := ParsePackageIdentity(instanceKey)
	if err != nil {
		return instanceKey
	}
	if id.IsProtocolRef {
		return instanceKey
	}
	return id.Name + "@" + id.BaseVersion
}

func packageIDFromInstanceKey(instanceKey string) (graph.PackageID, error) {
	id, err := ParsePackageIdentity(instanceKey)
	if err != nil {
		return graph.PackageID{}, err
	}
	if id.IsProtocolRef {
		return graph.PackageID{Name: id.Name, Version: id.BaseVersion}, nil
	}
	pkgID := graph.PackageID{Name: id.Name, Version: id.BaseVersion}
	if id.PeerSuffix != "" {
		ppc, err := peerSuffixToProviders(id.PeerSuffix)
		if err != nil {
			return graph.PackageID{}, apperr.New(apperr.Lockfile, "pnpm.graph", instanceKey, err.Error())
		}
		pkgID.PeerProviderContext = ppc
	}
	pkgID.Normalize()
	return pkgID, nil
}

func instanceKeyToGraphKey(instanceKey string) (string, error) {
	id, err := packageIDFromInstanceKey(instanceKey)
	if err != nil {
		return "", err
	}
	return id.Key(), nil
}

func graphKeyToInstanceKey(graphKey string) (string, error) {
	if isProtocolRef(graphKey) {
		return graphKey, nil
	}
	base, peerPart, hasPeer := strings.Cut(graphKey, "#")
	if !hasPeer {
		return graphKey, nil
	}
	return base + "(" + peerPart + ")", nil
}

func peerSuffixToProviders(suffix string) (graph.PeerProviderContext, error) {
	if suffix == "" {
		return nil, nil
	}
	if !strings.HasPrefix(suffix, "(") || !strings.HasSuffix(suffix, ")") {
		return nil, fmt.Errorf("malformed peer suffix %q", suffix)
	}
	inner := suffix[1 : len(suffix)-1]
	if inner == "" {
		return nil, fmt.Errorf("empty peer suffix")
	}
	var ppc graph.PeerProviderContext
	for _, part := range strings.Split(inner, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		name, ver := splitNameVersionKey(part)
		if name == "" || ver == "" {
			return nil, fmt.Errorf("malformed peer provider %q", part)
		}
		ppc = append(ppc, graph.PeerProvider{
			Name:    name,
			Version: ver,
			Key:     part,
		})
	}
	ppc.Sort()
	return ppc, nil
}
