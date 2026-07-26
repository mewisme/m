package graph

// SchemaVersion is the version field on serialized Graph documents.
const SchemaVersion = 1

// CacheSchemaVersion versions internal resolve/cache blobs.
// It is independent of public lockfile formats (m.lock / adapters).
const CacheSchemaVersion = 1

// DepKind classifies a dependency edge.
type DepKind string

const (
	DepProd     DepKind = "prod"
	DepDev      DepKind = "dev"
	DepOptional DepKind = "optional"
	DepPeer     DepKind = "peer"
)

// Edge is a directed dependency from an importer or package to a package key.
type Edge struct {
	From     string  `json:"from"` // importer id or package key
	To       string  `json:"to"`   // package key
	Kind     DepKind `json:"kind"`
	Range    string  `json:"range,omitempty"`
	Optional bool    `json:"optional,omitempty"`
}

// Package is a resolved package snapshot in the canonical graph.
type Package struct {
	ID         PackageID `json:"id"`
	Integrity  string    `json:"integrity,omitempty"`
	TarballURL string    `json:"tarballUrl,omitempty"`
}

// Importer is a workspace package that requests dependencies.
type Importer struct {
	ID   ImporterID `json:"id"`
	Name string     `json:"name,omitempty"`
	Path string     `json:"path,omitempty"` // relative path; usually same as ID
}

// Graph is the immutable canonical dependency graph after Validate.
type Graph struct {
	SchemaVersion int        `json:"schemaVersion"`
	Importers     []Importer `json:"importers"`
	Packages      []Package  `json:"packages"`
	Edges         []Edge     `json:"edges"`
}
