package snapshot

import (
	"fmt"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/lockfile/mlock"
	"github.com/mewisme/m/internal/manifest"
)

// ValidateRestorePair checks snapshot metadata, manifest/lock consistency, and graph digest.
func ValidateRestorePair(rec Record) (*graph.Graph, []byte, error) {
	if rec.Meta == nil {
		return nil, nil, apperr.New(apperr.Internal, "snapshot.validate", "snapshot", "nil metadata")
	}
	if err := rec.Meta.Normalize(); err != nil {
		return nil, nil, err
	}
	if len(rec.Manifest) == 0 {
		return nil, nil, apperr.New(apperr.Internal, "snapshot.validate", "package.json", "empty manifest")
	}
	if len(rec.Lock) == 0 {
		return nil, nil, apperr.New(apperr.Internal, "snapshot.validate", "m.lock", "empty lock")
	}

	doc, err := manifest.Parse(rec.Manifest)
	if err != nil {
		return nil, nil, err
	}
	norm, err := manifest.ToNormalized(doc)
	if err != nil {
		return nil, nil, err
	}

	lockDoc, err := mlock.Decode(rec.Lock)
	if err != nil {
		return nil, nil, err
	}
	g, err := mlock.ToGraph(lockDoc)
	if err != nil {
		return nil, nil, err
	}

	manifestSpecs := map[graph.ImporterID][]mlock.Specifier{
		graph.RootImporter: mlock.SpecifiersFromManifest(norm),
	}
	if drift := mlock.ValidateFrozen(lockDoc, manifestSpecs); len(drift) > 0 {
		return nil, nil, apperr.New(apperr.Lockfile, "snapshot.validate", "m.lock",
			fmt.Sprintf("manifest/lock pair drift:\n%s", mlock.FormatDrift(drift)))
	}

	digest, err := GraphDigest(g)
	if err != nil {
		return nil, nil, err
	}
	if rec.Meta.GraphDigest == "" {
		return nil, nil, apperr.New(apperr.Internal, "snapshot.validate", "snapshot", "missing graphDigest")
	}
	if digest != rec.Meta.GraphDigest {
		return nil, nil, apperr.New(apperr.Lockfile, "snapshot.validate", "snapshot",
			fmt.Sprintf("graph digest mismatch: recorded %s recomputed %s", rec.Meta.GraphDigest, digest))
	}

	return g, rec.Manifest, nil
}
