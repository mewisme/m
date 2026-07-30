package binmeta

import (
	"bytes"
	"encoding/json"
	"os"

	"github.com/mewisme/mew/internal/apperr"
)

// Read loads bins metadata from node_modules/.mew/bins.v1.json.
// Missing file returns (nil, nil).
func Read(nodeModules string) (*Document, error) {
	path := Path(nodeModules)
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, apperr.Wrap(apperr.IO, "binmeta.read", path, err)
	}
	var doc Document
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, apperr.Wrap(apperr.Integrity, "binmeta.read", path, err)
	}
	if doc.SchemaVersion != SchemaVersion {
		return nil, apperr.New(apperr.Integrity, "binmeta.read", path, "unsupported schema version")
	}
	return &doc, nil
}

// Write encodes and writes bins metadata atomically to the staging path.
func Write(nodeModules string, doc *Document) error {
	if doc == nil {
		return apperr.New(apperr.Internal, "binmeta.write", nodeModules, "nil document")
	}
	doc.SchemaVersion = SchemaVersion
	SortRecords(doc.Records)
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return apperr.Wrap(apperr.Internal, "binmeta.write", nodeModules, err)
	}
	dir := Dir(nodeModules)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return apperr.Wrap(apperr.IO, "binmeta.write", dir, err)
	}
	path := Path(nodeModules)
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

// Validate checks document invariants before publish.
func Validate(doc *Document) error {
	if doc == nil {
		return apperr.New(apperr.Internal, "binmeta.validate", "", "nil document")
	}
	if doc.SchemaVersion != SchemaVersion {
		return apperr.New(apperr.Integrity, "binmeta.validate", "", "unsupported schema version")
	}
	if doc.GenerationID == "" {
		return apperr.New(apperr.Integrity, "binmeta.validate", "", "missing generationID")
	}
	if doc.Fingerprint == "" {
		return apperr.New(apperr.Integrity, "binmeta.validate", "", "missing fingerprint")
	}
	seen := map[string]struct{}{}
	seenBin := map[string]struct{}{}
	for _, rec := range doc.Records {
		if rec.DeclaredBin == "" || rec.MaterializedShim == "" {
			return apperr.New(apperr.Integrity, "binmeta.validate", rec.DeclaredBin, "incomplete record")
		}
		if _, dup := seenBin[rec.DeclaredBin]; dup {
			return apperr.New(apperr.Integrity, "binmeta.validate", rec.DeclaredBin, "duplicate bin command")
		}
		seenBin[rec.DeclaredBin] = struct{}{}
		key := rec.DeclaredBin + "\x00" + rec.DependencyName + "\x00" + rec.ResolvedPackage
		if _, dup := seen[key]; dup {
			return apperr.New(apperr.Integrity, "binmeta.validate", rec.DeclaredBin, "duplicate record")
		}
		seen[key] = struct{}{}
	}
	return nil
}

// GenerationMatches reports whether doc matches the expected generation and fingerprint.
func GenerationMatches(doc *Document, generationID, fingerprint string) bool {
	if doc == nil {
		return false
	}
	return doc.GenerationID == generationID && doc.Fingerprint == fingerprint
}

// Stale reports whether doc exists but does not match the current generation binding.
func Stale(doc *Document, generationID, fingerprint string) bool {
	if doc == nil {
		return false
	}
	return !GenerationMatches(doc, generationID, fingerprint)
}
