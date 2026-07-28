package manifest

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/graph"
)

// SchemaVersion versions serialized Manifest documents.
const SchemaVersion = 1

// DepKind aliases graph dependency kinds for manifest declarations.
type DepKind = graph.DepKind

const (
	DepProd     = graph.DepProd
	DepDev      = graph.DepDev
	DepOptional = graph.DepOptional
	DepPeer     = graph.DepPeer
)

// Dependency is a declared dependency from package.json.
type Dependency struct {
	Name  string  `json:"name"`
	Range string  `json:"range"`
	Kind  DepKind `json:"kind"`
}

// Manifest is the normalized package.json view used by resolve.
type Manifest struct {
	SchemaVersion int               `json:"schemaVersion"`
	Name          string            `json:"name,omitempty"`
	Version       string            `json:"version,omitempty"`
	Dependencies  []Dependency      `json:"dependencies"`
	Overrides     map[string]string `json:"overrides,omitempty"` // flattened override path → specifier
}

// ParseJSON unmarshals a Manifest document and normalizes it.
func ParseJSON(data []byte) (*Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, apperr.Wrap(apperr.Config, "manifest.parse", "manifest", err)
	}
	if err := m.Normalize(); err != nil {
		return nil, err
	}
	return &m, nil
}

// Normalize sorts dependencies and fills schema version.
func (m *Manifest) Normalize() error {
	if m == nil {
		return apperr.New(apperr.Config, "manifest.normalize", "manifest", "nil manifest")
	}
	if m.SchemaVersion == 0 {
		m.SchemaVersion = SchemaVersion
	}
	if m.SchemaVersion != SchemaVersion {
		return apperr.New(apperr.Config, "manifest.normalize", "manifest",
			fmt.Sprintf("unsupported schemaVersion %d", m.SchemaVersion))
	}
	if m.Dependencies == nil {
		m.Dependencies = []Dependency{}
	}
	sort.SliceStable(m.Dependencies, func(i, j int) bool {
		a, b := m.Dependencies[i], m.Dependencies[j]
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.Name != b.Name {
			return a.Name < b.Name
		}
		return a.Range < b.Range
	})
	seen := make(map[string]struct{}, len(m.Dependencies))
	for _, d := range m.Dependencies {
		if d.Name == "" {
			return apperr.New(apperr.Config, "manifest.normalize", "manifest", "empty dependency name")
		}
		key := string(d.Kind) + ":" + d.Name
		if _, ok := seen[key]; ok {
			return apperr.New(apperr.Config, "manifest.normalize", "manifest",
				fmt.Sprintf("duplicate dependency %s %q", d.Kind, d.Name))
		}
		seen[key] = struct{}{}
	}
	return nil
}

// EncodeJSON normalizes then encodes with indent and trailing newline.
func EncodeJSON(m *Manifest) ([]byte, error) {
	if err := m.Normalize(); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, apperr.Wrap(apperr.Config, "manifest.encode", "manifest", err)
	}
	return append(data, '\n'), nil
}
