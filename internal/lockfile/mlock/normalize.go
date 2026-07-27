package mlock

import (
	"fmt"
	"sort"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/policy"
)

// Normalize sorts collections and validates required fields.
func (d *Document) Normalize() error {
	if d == nil {
		return apperr.New(apperr.Lockfile, "mlock.normalize", "m.lock", "nil document")
	}
	if d.LockfileVersion == 0 {
		d.LockfileVersion = LockfileVersion
	}
	if err := d.Settings.Normalize(); err != nil {
		return err
	}
	if d.Importers == nil {
		d.Importers = []ImporterSection{}
	}
	if d.Packages == nil {
		d.Packages = []graph.Package{}
	}
	if d.Edges == nil {
		d.Edges = []graph.Edge{}
	}

	for i := range d.Packages {
		d.Packages[i].ID.Normalize()
	}

	sort.SliceStable(d.Importers, func(i, j int) bool {
		return string(d.Importers[i].ID) < string(d.Importers[j].ID)
	})
	for i := range d.Importers {
		if err := normalizeSpecifiers(&d.Importers[i]); err != nil {
			return err
		}
	}
	sort.SliceStable(d.Packages, func(i, j int) bool {
		return d.Packages[i].ID.Key() < d.Packages[j].ID.Key()
	})
	sort.SliceStable(d.Edges, func(i, j int) bool {
		a, b := d.Edges[i], d.Edges[j]
		if a.From != b.From {
			return a.From < b.From
		}
		if a.Name != b.Name {
			return a.Name < b.Name
		}
		if a.To != b.To {
			return a.To < b.To
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		return a.Range < b.Range
	})

	importerSeen := make(map[graph.ImporterID]struct{}, len(d.Importers))
	for _, im := range d.Importers {
		if im.ID == "" {
			return apperr.New(apperr.Lockfile, "mlock.normalize", "m.lock", "empty importer id")
		}
		if _, ok := importerSeen[im.ID]; ok {
			return apperr.New(apperr.Lockfile, "mlock.normalize", "m.lock",
				fmt.Sprintf("duplicate importer %q", im.ID))
		}
		importerSeen[im.ID] = struct{}{}
	}

	pkgByKey := make(map[string]graph.Package, len(d.Packages))
	for _, p := range d.Packages {
		if p.ID.Name == "" || p.ID.Version == "" {
			return apperr.New(apperr.Lockfile, "mlock.normalize", "m.lock", "package missing name or version")
		}
		key := p.ID.Key()
		if prev, ok := pkgByKey[key]; ok {
			if prev.Integrity != p.Integrity || prev.TarballURL != p.TarballURL {
				return apperr.New(apperr.Lockfile, "mlock.normalize", "peer-context",
					fmt.Sprintf("peer-context identity collision for %q", key))
			}
			return apperr.New(apperr.Lockfile, "mlock.normalize", "peer-context",
				fmt.Sprintf("duplicate package key %q", key))
		}
		pkgByKey[key] = p
	}

	for i := range d.Edges {
		graph.NormalizeEdge(&d.Edges[i])
		e := d.Edges[i]
		if e.Name == "" {
			return apperr.New(apperr.Lockfile, "mlock.normalize", "m.lock", "edge missing name")
		}
		if e.To == "" {
			return apperr.New(apperr.Lockfile, "mlock.normalize", "m.lock", "edge missing to")
		}
		if _, ok := pkgByKey[e.To]; !ok {
			return apperr.New(apperr.Lockfile, "mlock.normalize", "m.lock",
				fmt.Sprintf("dangling edge to %q", e.To))
		}
		if e.From == "" {
			return apperr.New(apperr.Lockfile, "mlock.normalize", "m.lock", "edge missing from")
		}
		if _, ok := pkgByKey[e.From]; !ok {
			if _, ok := importerSeen[graph.ImporterID(e.From)]; !ok {
				return apperr.New(apperr.Lockfile, "mlock.normalize", "m.lock",
					fmt.Sprintf("edge from unknown %q", e.From))
			}
		}
	}
	return nil
}

func normalizeSpecifiers(im *ImporterSection) error {
	if im.Specifiers == nil {
		im.Specifiers = []Specifier{}
	}
	sort.SliceStable(im.Specifiers, func(i, j int) bool {
		a, b := im.Specifiers[i], im.Specifiers[j]
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.Name != b.Name {
			return a.Name < b.Name
		}
		return a.Range < b.Range
	})
	seen := make(map[string]struct{}, len(im.Specifiers))
	for _, s := range im.Specifiers {
		if s.Name == "" {
			return apperr.New(apperr.Lockfile, "mlock.normalize", "m.lock", "empty specifier name")
		}
		key := string(s.Kind) + ":" + s.Name
		if _, ok := seen[key]; ok {
			return apperr.New(apperr.Lockfile, "mlock.normalize", "m.lock",
				fmt.Sprintf("duplicate specifier %s %q", s.Kind, s.Name))
		}
		seen[key] = struct{}{}
	}
	return nil
}

// Normalize fills defaults on settings.
func (s *Settings) Normalize() error {
	if s == nil {
		return apperr.New(apperr.Lockfile, "mlock.normalize", "settings", "nil settings")
	}
	if s.Linker == "" {
		s.Linker = "auto"
	}
	switch s.Linker {
	case "auto", "hoisted", "isolated":
	default:
		return apperr.New(apperr.Lockfile, "mlock.normalize", "settings",
			fmt.Sprintf("unknown linker %q", s.Linker))
	}
	return s.Policy.Normalize()
}

// DefaultSettings returns canonical v1 settings.
func DefaultSettings() Settings {
	pol := policy.Policy{StrictPeerDependencies: true}
	_ = pol.Normalize()
	return Settings{Linker: "auto", Policy: pol}
}
