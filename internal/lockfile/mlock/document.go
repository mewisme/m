package mlock

import (
	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/lockfile"
	"github.com/mewisme/m/internal/policy"
)

// LockfileVersion is the current m.lock schema version.
const LockfileVersion = 3

// Document is the on-disk m.lock v1 shape.
type Document struct {
	LockfileVersion int                 `json:"lockfileVersion"`
	Checksum        string              `json:"checksum"`
	Settings        Settings            `json:"settings"`
	Importers       []ImporterSection   `json:"importers"`
	Packages        []graph.Package     `json:"packages"`
	Edges           []graph.Edge        `json:"edges"`
	Extensions      lockfile.Extensions `json:"extensions,omitempty"`
}

// Settings snapshots linker and policy for install handoff.
type Settings struct {
	Linker                    string        `json:"linker"`
	Policy                    policy.Policy `json:"policy"`
	OverridesFingerprint      string        `json:"overridesFingerprint,omitempty"`
	ResolverPolicyFingerprint string        `json:"resolverPolicyFingerprint,omitempty"`
	TargetPlatformFingerprint string        `json:"targetPlatformFingerprint,omitempty"`
}

// ImporterSection is one workspace package with declared specifiers.
type ImporterSection struct {
	ID         graph.ImporterID `json:"id"`
	Name       string           `json:"name,omitempty"`
	Path       string           `json:"path,omitempty"`
	Specifiers []Specifier      `json:"specifiers"`
}

// Specifier is a declared dependency from package.json.
type Specifier struct {
	Name  string        `json:"name"`
	Range string        `json:"range"`
	Kind  graph.DepKind `json:"kind"`
}
