package pnpm

import (
	"strings"

	"github.com/mewisme/mew/internal/manifest"
)

// ParseAliasFromImporterDep extracts the actual package name when depName is an npm alias.
func ParseAliasFromImporterDep(aliasName, specifier, versionRef string) (actualName string, resolutionRef string, isAlias bool) {
	spec := strings.TrimSpace(specifier)
	ver := strings.TrimSpace(versionRef)
	if strings.HasPrefix(spec, "npm:") {
		sp, err := manifest.ParseSpecifier(aliasName, spec)
		if err == nil && sp.Protocol == manifest.ProtocolNpm {
			ref := ver
			if ref == "" {
				ref = sp.TargetName + "@" + sp.Range
			}
			return sp.TargetName, ref, true
		}
	}
	if strings.HasPrefix(ver, "npm:") {
		sp, err := manifest.ParseSpecifier(aliasName, ver)
		if err == nil && sp.Protocol == manifest.ProtocolNpm {
			return sp.TargetName, ver, true
		}
	}
	// Lock version field uses actual package@version while dep key is the alias label.
	if ver != "" && strings.Contains(ver, "@") && !strings.Contains(ver, "(") {
		if id, err := ParsePackageIdentity(ver); err == nil && !id.IsProtocolRef && id.PeerSuffix == "" && id.Name != aliasName {
			return id.Name, ver, true
		}
	}
	return aliasName, ver, false
}
