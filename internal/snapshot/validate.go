package snapshot

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/lockfile/mlock"
	"github.com/mewisme/mew/internal/manifest"
	"github.com/mewisme/mew/internal/resolver"
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
	seenMember := map[string]string{}
	for rel, raw := range rec.MemberManifests {
		rel = filepath.ToSlash(rel)
		id, err := ParseMemberManifestPath(rel)
		if err != nil {
			return nil, nil, err
		}
		fold := strings.ToLower(rel)
		if prev, ok := seenMember[fold]; ok && prev != rel {
			return nil, nil, apperr.New(apperr.IO, "snapshot.validate", rel, "duplicate member manifest path")
		}
		seenMember[fold] = rel
		memDoc, err := manifest.Parse(raw)
		if err != nil {
			return nil, nil, err
		}
		memNorm, err := manifest.ToNormalized(memDoc)
		if err != nil {
			return nil, nil, err
		}
		if !lockImporterExists(lockDoc, id) {
			return nil, nil, apperr.New(apperr.Lockfile, "snapshot.validate", string(id), "member manifest without lock importer")
		}
		manifestSpecs[id] = mlock.SpecifiersFromManifest(memNorm)
	}
	if rec.Meta.SchemaVersion >= SchemaVersion {
		for _, im := range lockDoc.Importers {
			if im.ID == graph.RootImporter {
				continue
			}
			rel := filepath.ToSlash(filepath.Join(string(im.ID), "package.json"))
			if _, ok := rec.MemberManifests[rel]; !ok {
				return nil, nil, apperr.New(apperr.Lockfile, "snapshot.validate", string(im.ID), "lock importer missing member manifest in snapshot")
			}
		}
	}
	if drift := mlock.ValidateFrozen(lockDoc, manifestSpecs); len(drift) > 0 {
		return nil, nil, apperr.New(apperr.Lockfile, "snapshot.validate", "m.lock",
			fmt.Sprintf("manifest/lock pair drift:\n%s", mlock.FormatDrift(drift)))
	}

	if locals, err := resolver.DecodeLocalSources(lockDoc.Extensions); err != nil {
		return nil, nil, err
	} else if len(locals) > 0 {
		pkgs := map[string]struct{}{}
		for _, p := range g.Packages {
			pkgs[p.ID.Key()] = struct{}{}
		}
		for key := range locals {
			if _, ok := pkgs[key]; !ok {
				return nil, nil, apperr.New(apperr.Lockfile, "snapshot.validate", key, "local extension references missing package")
			}
		}
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

func lockImporterExists(doc *mlock.Document, id graph.ImporterID) bool {
	if doc == nil {
		return false
	}
	for _, im := range doc.Importers {
		if im.ID == id {
			return true
		}
	}
	return false
}
