package mlock

import (
	"encoding/json"
	"fmt"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/graph"
)

const lockfileV1 = 1
const lockfileV2 = 2

// v1PeerRef is the deprecated range-based peer identity from lockfile v1.
type v1PeerRef struct {
	Name  string `json:"name"`
	Range string `json:"range"`
}

// v1PackageID detects v1 peer-range identity embedded in packages.
type v1PackageID struct {
	Name        string      `json:"name"`
	Version     string      `json:"version"`
	PeerContext []v1PeerRef `json:"peerContext,omitempty"`
}

// Migrate upgrades doc to the current lockfile version.
func Migrate(doc *Document) error {
	if doc == nil {
		return apperr.New(apperr.Lockfile, "mlock.migrate", "m.lock", "nil document")
	}
	if doc.LockfileVersion == 0 {
		doc.LockfileVersion = LockfileVersion
	}
	switch doc.LockfileVersion {
	case LockfileVersion:
		return nil
	case lockfileV2:
		migrateEdgesV2ToV3(doc)
		doc.LockfileVersion = LockfileVersion
		return nil
	case lockfileV1:
		return rejectV1PeerRangeLock(doc)
	default:
		return apperr.New(apperr.Lockfile, "mlock.migrate", "m.lock",
			fmt.Sprintf("unsupported lockfileVersion %d", doc.LockfileVersion))
	}
}

func rejectV1PeerRangeLock(doc *Document) error {
	if hasV1PeerRangeIdentity(doc.Packages) {
		return apperr.New(apperr.Lockfile, "mlock.migrate", "m.lock",
			"lockfile v1 uses range-based peerContext; re-resolve with m lock to upgrade to lockfileVersion 2 (peerProviders)")
	}
	return apperr.New(apperr.Lockfile, "mlock.migrate", "m.lock",
		"lockfile v1 is no longer supported; run m lock to regenerate lockfileVersion 2")
}

func hasV1PeerRangeIdentity(packages []graph.Package) bool {
	for _, p := range packages {
		raw, err := json.Marshal(p.ID)
		if err != nil {
			continue
		}
		var v1 v1PackageID
		if err := json.Unmarshal(raw, &v1); err != nil {
			continue
		}
		for _, ref := range v1.PeerContext {
			if ref.Range != "" {
				return true
			}
		}
	}
	return false
}

func migrateEdgesV2ToV3(doc *Document) {
	for i := range doc.Edges {
		graph.NormalizeEdge(&doc.Edges[i])
	}
}
