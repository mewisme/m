package mlock

import (
	"fmt"
	"strings"

	"github.com/mewisme/mew/internal/graph"
)

// DriftKind classifies manifest vs lock specifier drift.
type DriftKind string

const (
	DriftAdded   DriftKind = "added"
	DriftRemoved DriftKind = "removed"
	DriftChanged DriftKind = "changed"
)

// DriftItem is one specifier difference for an importer.
type DriftItem struct {
	Importer      graph.ImporterID
	Kind          DriftKind
	Name          string
	DepKind       graph.DepKind
	LockRange     string
	ManifestRange string
}

func (d DriftItem) String() string {
	switch d.Kind {
	case DriftAdded:
		return fmt.Sprintf("%s: added %s %s (manifest %s)", d.Importer, d.DepKind, d.Name, d.ManifestRange)
	case DriftRemoved:
		return fmt.Sprintf("%s: removed %s %s (lock had %s)", d.Importer, d.DepKind, d.Name, d.LockRange)
	case DriftChanged:
		return fmt.Sprintf("%s: changed %s %s lock %s → manifest %s", d.Importer, d.DepKind, d.Name, d.LockRange, d.ManifestRange)
	default:
		return fmt.Sprintf("%s: %s %s", d.Importer, d.DepKind, d.Name)
	}
}

type specKey struct {
	kind graph.DepKind
	name string
}

// CompareSpecifiers returns drift between lock importers and manifest specifiers.
func CompareSpecifiers(lock []ImporterSection, manifest map[graph.ImporterID][]Specifier) []DriftItem {
	var out []DriftItem

	lockByID := make(map[graph.ImporterID]map[specKey]string, len(lock))
	for _, im := range lock {
		m := make(map[specKey]string, len(im.Specifiers))
		for _, s := range im.Specifiers {
			m[specKey{kind: s.Kind, name: s.Name}] = s.Range
		}
		lockByID[im.ID] = m
	}

	seen := make(map[graph.ImporterID]struct{})
	for id, specs := range manifest {
		seen[id] = struct{}{}
		lockSpecs := lockByID[id]
		if lockSpecs == nil {
			lockSpecs = map[specKey]string{}
		}
		man := make(map[specKey]string, len(specs))
		for _, s := range specs {
			man[specKey{kind: s.Kind, name: s.Name}] = s.Range
		}
		for k, rng := range man {
			if lockRng, ok := lockSpecs[k]; !ok {
				out = append(out, DriftItem{
					Importer: id, Kind: DriftAdded, Name: k.name, DepKind: k.kind, ManifestRange: rng,
				})
			} else if lockRng != rng {
				out = append(out, DriftItem{
					Importer: id, Kind: DriftChanged, Name: k.name, DepKind: k.kind,
					LockRange: lockRng, ManifestRange: rng,
				})
			}
		}
		for k, rng := range lockSpecs {
			if _, ok := man[k]; !ok {
				out = append(out, DriftItem{
					Importer: id, Kind: DriftRemoved, Name: k.name, DepKind: k.kind, LockRange: rng,
				})
			}
		}
	}
	for _, im := range lock {
		if _, ok := seen[im.ID]; ok {
			continue
		}
		for _, s := range im.Specifiers {
			out = append(out, DriftItem{
				Importer: im.ID, Kind: DriftRemoved, Name: s.Name, DepKind: s.Kind, LockRange: s.Range,
			})
		}
	}
	return out
}

// FormatDrift returns a human-readable drift summary.
func FormatDrift(items []DriftItem) string {
	if len(items) == 0 {
		return ""
	}
	lines := make([]string, len(items))
	for i, d := range items {
		lines[i] = d.String()
	}
	return strings.Join(lines, "\n")
}

// ValidateFrozen reports drift between doc importers and manifest specifiers.
func ValidateFrozen(doc *Document, manifest map[graph.ImporterID][]Specifier) []DriftItem {
	if doc == nil {
		return nil
	}
	return CompareSpecifiers(doc.Importers, manifest)
}
