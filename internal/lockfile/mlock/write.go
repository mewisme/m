package mlock

import (
	"path/filepath"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/fsx"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/resolver"
)

// FromResolution builds a lock document from a resolve result.
func FromResolution(res *resolver.Resolution, specifiers map[graph.ImporterID][]Specifier, settings Settings) (*Document, error) {
	if res == nil || res.Graph == nil {
		return nil, apperr.New(apperr.Lockfile, "mlock.write", "resolution", "nil resolution graph")
	}
	doc, err := FromGraph(res.Graph, specifiers, settings)
	if err != nil {
		return nil, err
	}
	if len(res.Extensions) > 0 {
		doc.Extensions = res.Extensions
	}
	return doc, nil
}

// WriteAtomic encodes doc and durably replaces path.
func WriteAtomic(path string, doc *Document) error {
	data, err := Encode(doc)
	if err != nil {
		return err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return apperr.Wrap(apperr.IO, "mlock.write", path, err)
	}
	if err := fsx.PublishFileDurable(abs, data, 0o644); err != nil {
		return apperr.Wrap(apperr.IO, "mlock.write", abs, err)
	}
	return nil
}

// SettingsFromEffective builds lock settings from effective config.
func SettingsFromEffective(eff *config.Effective) (Settings, error) {
	s := DefaultSettings()
	if eff == nil {
		return s, nil
	}
	if v, err := config.Get(eff, "install.linker"); err == nil {
		if linker, ok := v.Raw.(string); ok && linker != "" {
			s.Linker = linker
		}
	}
	if pol := resolver.PolicyFromEffective(eff); pol != nil {
		s.Policy = *pol
	}
	return s, s.Normalize()
}

// SettingsWithFingerprints builds lock settings including resolver fingerprint snapshots.
func SettingsWithFingerprints(eff *config.Effective, overrides map[string]string) (Settings, error) {
	s, err := SettingsFromEffective(eff)
	if err != nil {
		return s, err
	}
	s.OverridesFingerprint = resolver.OverridesFingerprint(overrides)
	s.ResolverPolicyFingerprint = resolver.PolicyFingerprint(resolver.PolicyFromEffective(eff))
	s.TargetPlatformFingerprint = resolver.TargetPlatformFingerprint(resolver.CurrentTarget())
	return s, nil
}
