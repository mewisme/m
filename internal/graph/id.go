package graph

import (
	"sort"
	"strings"
)

// ImporterID is a POSIX-style path relative to the project root.
// The root importer is ".".
type ImporterID string

// RootImporter is the workspace/project root importer id.
const RootImporter ImporterID = "."

// PeerProvider is a resolved peer dependency provider in package identity.
type PeerProvider struct {
	Name    string `json:"name"`    // peer package name
	Version string `json:"version"` // resolved provider version
	Key     string `json:"key"`     // resolved provider package key
}

// PeerProviderContext is a sorted set of resolved peer providers in package identity.
type PeerProviderContext []PeerProvider

// Sort orders providers by name then version for deterministic keys and encoding.
func (ppc PeerProviderContext) Sort() {
	sort.SliceStable(ppc, func(i, j int) bool {
		if ppc[i].Name != ppc[j].Name {
			return ppc[i].Name < ppc[j].Name
		}
		return ppc[i].Version < ppc[j].Version
	})
}

// String returns the peer suffix used in PackageID.Key (empty when no providers).
func (ppc PeerProviderContext) String() string {
	if len(ppc) == 0 {
		return ""
	}
	parts := make([]string, len(ppc))
	for i, p := range ppc {
		if p.Key != "" {
			parts[i] = p.Key
			continue
		}
		parts[i] = p.Name + "@" + p.Version
	}
	return strings.Join(parts, ",")
}

// PackageID identifies a resolved package, optionally under a peer provider context.
type PackageID struct {
	Name                string              `json:"name"`
	Version             string              `json:"version"`
	PeerProviderContext PeerProviderContext `json:"peerProviders,omitempty"`
}

// Key returns the stable package key:
//
//	name@version
//	name@version#providerKey1,providerKey2
//
// Providers in PeerProviderContext must already be sorted (Validate/Builder ensure this).
func (id PackageID) Key() string {
	base := id.Name + "@" + id.Version
	if suffix := id.PeerProviderContext.String(); suffix != "" {
		return base + "#" + suffix
	}
	return base
}

// Normalize sorts PeerProviderContext in place for Key stability.
func (id *PackageID) Normalize() {
	if id == nil {
		return
	}
	id.PeerProviderContext.Sort()
}
