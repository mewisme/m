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

// PeerRef is one peer dependency constraint in a peer context.
type PeerRef struct {
	Name  string `json:"name"`
	Range string `json:"range"`
}

// PeerContext is a sorted set of peer constraints that participate in package identity.
type PeerContext []PeerRef

// Sort orders peers by name then range for deterministic keys and encoding.
func (pc PeerContext) Sort() {
	sort.SliceStable(pc, func(i, j int) bool {
		if pc[i].Name != pc[j].Name {
			return pc[i].Name < pc[j].Name
		}
		return pc[i].Range < pc[j].Range
	})
}

// String returns the peer suffix used in PackageID.Key (empty when no peers).
func (pc PeerContext) String() string {
	if len(pc) == 0 {
		return ""
	}
	parts := make([]string, len(pc))
	for i, p := range pc {
		parts[i] = p.Name + "@" + p.Range
	}
	return strings.Join(parts, ",")
}

// PackageID identifies a resolved package, optionally under a peer context.
type PackageID struct {
	Name        string      `json:"name"`
	Version     string      `json:"version"`
	PeerContext PeerContext `json:"peerContext,omitempty"`
}

// Key returns the stable package key:
//
//	name@version
//	name@version#peer1@range1,peer2@range2
//
// Peers in PeerContext must already be sorted (Validate/Builder ensure this).
func (id PackageID) Key() string {
	base := id.Name + "@" + id.Version
	if suffix := id.PeerContext.String(); suffix != "" {
		return base + "#" + suffix
	}
	return base
}

// Normalize sorts PeerContext in place for Key stability.
func (id *PackageID) Normalize() {
	if id == nil {
		return
	}
	id.PeerContext.Sort()
}
