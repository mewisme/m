package resolver

import (
	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/manifest"
)

func (s *resolveState) rewriteCatalog(displayName, spec string, protocol manifest.Protocol, rng string) (string, manifest.Protocol, error) {
	if protocol != manifest.ProtocolCatalog {
		return rng, protocol, nil
	}
	if s.catalog == nil {
		return "", "", apperr.New(apperr.Manifest, "resolver.catalog", displayName, "catalog not defined")
	}
	entryKey := rng
	if entryKey == "" {
		entryKey = displayName
	}
	ver, err := s.catalog.ResolveEntry(entryKey)
	if err != nil {
		return "", "", apperr.Wrap(apperr.Manifest, "resolver.catalog", displayName, err)
	}
	return ver, manifest.ProtocolRegistry, nil
}
