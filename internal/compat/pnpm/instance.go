package pnpm

import (
	"fmt"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/graph"
)

func buildInstanceSet(doc *Document) []string {
	seen := make(map[string]struct{}, len(doc.Packages)+len(doc.Snapshots))
	for k := range doc.Packages {
		seen[k] = struct{}{}
	}
	for k := range doc.Snapshots {
		seen[k] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	return sortedStrings(out)
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
