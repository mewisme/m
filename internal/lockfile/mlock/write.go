package mlock

import (
	"os"
	"path/filepath"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/config"
	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/resolver"
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

// WriteAtomic encodes doc and atomically replaces path.
func WriteAtomic(path string, doc *Document) error {
	data, err := Encode(doc)
	if err != nil {
		return err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return apperr.Wrap(apperr.IO, "mlock.write", path, err)
	}
	dir := filepath.Dir(abs)
	tmp, err := os.CreateTemp(dir, ".m.lock.*.tmp")
	if err != nil {
		return apperr.Wrap(apperr.IO, "mlock.write", abs, err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return apperr.Wrap(apperr.IO, "mlock.write", abs, err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return apperr.Wrap(apperr.IO, "mlock.write", abs, err)
	}
	if err := tmp.Close(); err != nil {
		return apperr.Wrap(apperr.IO, "mlock.write", abs, err)
	}
	if err := os.Rename(tmpName, abs); err != nil {
		_ = os.Remove(abs)
		if err2 := os.Rename(tmpName, abs); err2 != nil {
			return apperr.Wrap(apperr.IO, "mlock.write", abs, err2)
		}
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
	s.Policy.AutoInstallPeers = config.Bool(eff, "resolve.autoInstallPeers", false)
	s.Policy.StrictPeerDependencies = config.Bool(eff, "resolve.strictPeerDependencies", true)
	return s, s.Normalize()
}
