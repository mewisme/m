package mlock

import (
	"bytes"
	"encoding/json"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/lockfile"
)

var knownTopLevel = map[string]struct{}{
	"lockfileVersion": {},
	"checksum":        {},
	"settings":        {},
	"importers":       {},
	"packages":        {},
	"edges":           {},
	"extensions":      {},
}

// Decode parses, migrates, normalizes, and verifies checksum.
func Decode(data []byte) (*Document, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, apperr.Wrap(apperr.Lockfile, "mlock.decode", "m.lock", err)
	}

	doc := &Document{}
	originalVersion := 0
	if v, ok := raw["lockfileVersion"]; ok {
		if err := json.Unmarshal(v, &doc.LockfileVersion); err != nil {
			return nil, apperr.Wrap(apperr.Lockfile, "mlock.decode", "lockfileVersion", err)
		}
		originalVersion = doc.LockfileVersion
	}
	if v, ok := raw["checksum"]; ok {
		if err := json.Unmarshal(v, &doc.Checksum); err != nil {
			return nil, apperr.Wrap(apperr.Lockfile, "mlock.decode", "checksum", err)
		}
	}
	if v, ok := raw["settings"]; ok {
		if err := json.Unmarshal(v, &doc.Settings); err != nil {
			return nil, apperr.Wrap(apperr.Lockfile, "mlock.decode", "settings", err)
		}
	}
	if v, ok := raw["importers"]; ok {
		if err := json.Unmarshal(v, &doc.Importers); err != nil {
			return nil, apperr.Wrap(apperr.Lockfile, "mlock.decode", "importers", err)
		}
	}
	if v, ok := raw["packages"]; ok {
		if doc.LockfileVersion == lockfileV1 && bytes.Contains(v, []byte(`"peerContext"`)) && bytes.Contains(v, []byte(`"range"`)) {
			return nil, apperr.New(apperr.Lockfile, "mlock.migrate", "m.lock",
				"lockfile v1 uses range-based peerContext; re-resolve with m lock to upgrade to lockfileVersion 2 (peerProviders)")
		}
		if err := json.Unmarshal(v, &doc.Packages); err != nil {
			return nil, apperr.Wrap(apperr.Lockfile, "mlock.decode", "packages", err)
		}
	}
	if v, ok := raw["edges"]; ok {
		if err := json.Unmarshal(v, &doc.Edges); err != nil {
			return nil, apperr.Wrap(apperr.Lockfile, "mlock.decode", "edges", err)
		}
	}

	ext := lockfile.Extensions{}
	if v, ok := raw["extensions"]; ok {
		if err := json.Unmarshal(v, &ext); err != nil {
			return nil, apperr.Wrap(apperr.Lockfile, "mlock.decode", "extensions", err)
		}
	}
	for k, v := range raw {
		if _, known := knownTopLevel[k]; !known {
			ext[k] = v
		}
	}
	if len(ext) > 0 {
		doc.Extensions = ext
	}

	if err := Migrate(doc); err != nil {
		return nil, err
	}
	if err := doc.Normalize(); err != nil {
		return nil, err
	}
	if originalVersion >= LockfileVersion {
		if err := verifyChecksum(doc); err != nil {
			return nil, err
		}
	}
	return doc, nil
}
