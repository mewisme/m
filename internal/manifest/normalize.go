package manifest

import (
	"github.com/mewisme/m/internal/apperr"
)

// ToNormalized flattens npm dependency maps into the resolver Manifest.
func ToNormalized(doc *Document) (*Manifest, error) {
	if doc == nil {
		return nil, apperr.New(apperr.Manifest, "manifest.normalize", "package.json", "nil document")
	}
	m := &Manifest{
		SchemaVersion: SchemaVersion,
		Name:          doc.Name,
		Version:       doc.Version,
		Dependencies:  nil,
	}
	add := func(kind DepKind, deps map[string]string) {
		for name, rng := range deps {
			m.Dependencies = append(m.Dependencies, Dependency{
				Name:  name,
				Range: rng,
				Kind:  kind,
			})
		}
	}
	add(DepProd, doc.Dependencies)
	add(DepDev, doc.DevDependencies)
	add(DepOptional, doc.OptionalDependencies)
	add(DepPeer, doc.PeerDependencies)
	if doc.Overrides != nil {
		flat, err := FlattenOverrides(doc.Overrides)
		if err != nil {
			return nil, err
		}
		m.Overrides = flat
	}
	if err := m.Normalize(); err != nil {
		return nil, err
	}
	return m, nil
}
