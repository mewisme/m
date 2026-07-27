package registry

import (
	"encoding/json"
)

// CacheSchemaVersion versions on-disk metadata cache entries.
const CacheSchemaVersion = 1

// PackageMetadata is registry metadata for a single version.
type PackageMetadata struct {
	Name       string
	Version    string
	Integrity  string
	TarballURL string
}

// Dist holds tarball location and integrity from a packument version.
type Dist struct {
	Integrity string `json:"integrity"`
	Tarball   string `json:"tarball"`
	Shasum    string `json:"shasum,omitempty"`
}

// PeerMeta is optional metadata for a peer dependency.
type PeerMeta struct {
	Optional bool `json:"optional,omitempty"`
}

// VersionMeta is one version entry inside a packument.
type VersionMeta struct {
	Name                 string              `json:"name,omitempty"`
	Version              string              `json:"version"`
	Deprecated           string              `json:"deprecated,omitempty"`
	Time                 string              `json:"time,omitempty"` // RFC3339 publish time when present
	Dist                 Dist                `json:"dist"`
	Dependencies         map[string]string   `json:"dependencies,omitempty"`
	DevDependencies      map[string]string   `json:"devDependencies,omitempty"`
	OptionalDependencies map[string]string   `json:"optionalDependencies,omitempty"`
	PeerDependencies     map[string]string   `json:"peerDependencies,omitempty"`
	PeerDependenciesMeta map[string]PeerMeta `json:"peerDependenciesMeta,omitempty"`
	OS                   []string            `json:"os,omitempty"`
	CPU                  []string            `json:"cpu,omitempty"`
	Libc                 []string            `json:"libc,omitempty"`
}

// Packument is an npm-compatible package metadata document.
type Packument struct {
	Name     string                 `json:"name"`
	DistTags map[string]string      `json:"dist-tags"`
	Versions map[string]VersionMeta `json:"versions"`
	Raw      json.RawMessage        `json:"-"`
}
