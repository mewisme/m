package bun

import (
	"encoding/json"

	"github.com/mewisme/mew/internal/lockfile"
)

const (
	FormatV1 = "bun-v1"
)

// Document is the parsed text bun.lock model.
type Document struct {
	LockfileVersion int                       `json:"lockfileVersion"`
	ConfigVersion   int                       `json:"configVersion,omitempty"`
	Workspaces      map[string]WorkspaceEntry `json:"workspaces,omitempty"`
	Packages        map[string]PackageArray   `json:"packages,omitempty"`
	Extensions      lockfile.Extensions       `json:"-"`
	Detection       lockfile.Detection        `json:"-"`
}

// WorkspaceEntry is one workspace importer block.
type WorkspaceEntry struct {
	Name                 string                     `json:"name,omitempty"`
	Dependencies         map[string]string          `json:"dependencies,omitempty"`
	DevDependencies      map[string]string          `json:"devDependencies,omitempty"`
	OptionalDependencies map[string]string          `json:"optionalDependencies,omitempty"`
	Extra                map[string]json.RawMessage `json:"-"`
}

// PackageArray is the bun.lock package tuple: [resolution, registry, info, integrity].
type PackageArray []json.RawMessage

// PackageInfo is the optional third element of a package tuple.
type PackageInfo struct {
	Dependencies         map[string]string `json:"dependencies,omitempty"`
	DevDependencies      map[string]string `json:"devDependencies,omitempty"`
	OptionalDependencies map[string]string `json:"optionalDependencies,omitempty"`
	PeerDependencies     map[string]string `json:"peerDependencies,omitempty"`
}
