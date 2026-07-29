package npm

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/lockfile"
)

var knownTopLevel = map[string]struct{}{
	"lockfileVersion": {},
	"name":            {},
	"requires":        {},
	"packages":        {},
	"dependencies":    {},
}

var knownPackageFields = map[string]struct{}{
	"name": {}, "version": {}, "resolved": {}, "integrity": {}, "link": {},
	"dev": {}, "devOptional": {}, "optional": {},
	"dependencies": {}, "devDependencies": {}, "optionalDependencies": {},
	"peerDependencies": {}, "bundledDependencies": {}, "workspaces": {},
	"license": {}, "engines": {}, "funding": {}, "cpu": {}, "os": {},
	"deprecated": {}, "bin": {}, "hasInstallScript": {},
}

var knownLegacyDepFields = map[string]struct{}{
	"version": {}, "resolved": {}, "integrity": {}, "requires": {}, "dependencies": {},
}

// Decode parses npm package-lock or shrinkwrap JSON.
func Decode(data []byte) (*Document, error) {
	if err := validateLockInput(data); err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, apperr.Wrap(apperr.Lockfile, "npm.decode", "package-lock.json", err)
	}
	if len(raw) > maxMapEntries {
		return nil, apperr.New(apperr.Lockfile, "npm.decode", "package-lock.json",
			fmt.Sprintf("exceeds %d top-level keys", maxMapEntries))
	}

	doc := &Document{
		Packages:     map[string]PackageEntry{},
		Dependencies: map[string]LegacyDep{},
		Extensions:   lockfile.Extensions{},
	}

	for k, v := range raw {
		if _, known := knownTopLevel[k]; known {
			continue
		}
		doc.Extensions[k] = append(json.RawMessage(nil), v...)
	}

	if verRaw, ok := raw["lockfileVersion"]; ok {
		var ver json.Number
		if err := json.Unmarshal(verRaw, &ver); err != nil {
			return nil, apperr.Wrap(apperr.Lockfile, "npm.decode", "lockfileVersion", err)
		}
		major, err := ver.Int64()
		if err != nil {
			return nil, apperr.Wrap(apperr.Lockfile, "npm.decode", "lockfileVersion", err)
		}
		doc.LockfileVersion = int(major)
	}

	if nameRaw, ok := raw["name"]; ok {
		_ = json.Unmarshal(nameRaw, &doc.Name)
	}
	if reqRaw, ok := raw["requires"]; ok {
		_ = json.Unmarshal(reqRaw, &doc.Requires)
	}

	if pkgRaw, ok := raw["packages"]; ok {
		var pkgMap map[string]json.RawMessage
		if err := json.Unmarshal(pkgRaw, &pkgMap); err != nil {
			return nil, apperr.Wrap(apperr.Lockfile, "npm.decode", "packages", err)
		}
		if len(pkgMap) > maxMapEntries {
			return nil, apperr.New(apperr.Lockfile, "npm.decode", "packages",
				fmt.Sprintf("exceeds %d package entries", maxMapEntries))
		}
		for path, entryRaw := range pkgMap {
			if err := validatePackagePath(path); err != nil {
				return nil, err
			}
			entry, err := decodePackageEntry(entryRaw)
			if err != nil {
				return nil, apperr.Wrap(apperr.Lockfile, "npm.decode", path, err)
			}
			doc.Packages[path] = entry
		}
	}

	if depRaw, ok := raw["dependencies"]; ok {
		var depMap map[string]json.RawMessage
		if err := json.Unmarshal(depRaw, &depMap); err != nil {
			return nil, apperr.Wrap(apperr.Lockfile, "npm.decode", "dependencies", err)
		}
		for name, entryRaw := range depMap {
			if err := validateDepName(name); err != nil {
				return nil, err
			}
			entry, err := decodeLegacyDep(entryRaw)
			if err != nil {
				return nil, apperr.Wrap(apperr.Lockfile, "npm.decode", name, err)
			}
			doc.Dependencies[name] = entry
		}
	}

	if err := rejectV1(doc); err != nil {
		return nil, err
	}
	if err := ValidateSupported(doc); err != nil {
		return nil, err
	}
	doc.Detection = DetectFromDocument(doc)
	return doc, nil
}

func rejectV1(doc *Document) error {
	if doc == nil {
		return apperr.New(apperr.Lockfile, "npm.decode", "document", "nil document")
	}
	if doc.LockfileVersion == 1 {
		return lockfile.NewUnsupported("npm.decode", "package-lock.json",
			"package-lock v1 is unsupported; regenerate with npm 7+ (lockfileVersion 2 or 3)")
	}
	if len(doc.Packages) == 0 && len(doc.Dependencies) > 0 {
		return lockfile.NewUnsupported("npm.decode", "package-lock.json",
			"nested-only lockfile (v1 layout) is unsupported; regenerate with npm 7+ (lockfileVersion 2 or 3)")
	}
	return nil
}

func decodePackageEntry(raw json.RawMessage) (PackageEntry, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return PackageEntry{}, err
	}
	var entry PackageEntry
	entry.Extra = map[string]json.RawMessage{}
	for k, v := range fields {
		switch k {
		case "name":
			_ = json.Unmarshal(v, &entry.Name)
		case "version":
			_ = json.Unmarshal(v, &entry.Version)
		case "resolved":
			_ = json.Unmarshal(v, &entry.Resolved)
		case "integrity":
			_ = json.Unmarshal(v, &entry.Integrity)
		case "link":
			_ = json.Unmarshal(v, &entry.Link)
		case "dev":
			_ = json.Unmarshal(v, &entry.Dev)
		case "devOptional":
			_ = json.Unmarshal(v, &entry.DevOptional)
		case "optional":
			_ = json.Unmarshal(v, &entry.Optional)
		case "dependencies":
			entry.Dependencies = decodeStringMap(v)
		case "devDependencies":
			entry.DevDependencies = decodeStringMap(v)
		case "optionalDependencies":
			entry.OptionalDependencies = decodeStringMap(v)
		case "peerDependencies":
			entry.PeerDependencies = decodeStringMap(v)
		case "bundledDependencies":
			_ = json.Unmarshal(v, &entry.BundledDependencies)
		case "workspaces":
			_ = json.Unmarshal(v, &entry.Workspaces)
		default:
			if _, known := knownPackageFields[k]; !known {
				entry.Extra[k] = append(json.RawMessage(nil), v...)
			}
		}
	}
	return entry, nil
}

func decodeLegacyDep(raw json.RawMessage) (LegacyDep, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return LegacyDep{}, err
	}
	var dep LegacyDep
	dep.Extra = map[string]json.RawMessage{}
	for k, v := range fields {
		switch k {
		case "version":
			_ = json.Unmarshal(v, &dep.Version)
		case "resolved":
			_ = json.Unmarshal(v, &dep.Resolved)
		case "integrity":
			_ = json.Unmarshal(v, &dep.Integrity)
		case "requires":
			_ = json.Unmarshal(v, &dep.Requires)
		case "dependencies":
			var nested map[string]json.RawMessage
			if err := json.Unmarshal(v, &nested); err != nil {
				return LegacyDep{}, err
			}
			dep.Deps = make(map[string]LegacyDep, len(nested))
			for name, childRaw := range nested {
				child, err := decodeLegacyDep(childRaw)
				if err != nil {
					return LegacyDep{}, err
				}
				dep.Deps[name] = child
			}
		default:
			if _, known := knownLegacyDepFields[k]; !known {
				dep.Extra[k] = append(json.RawMessage(nil), v...)
			}
		}
	}
	return dep, nil
}

func decodeStringMap(raw json.RawMessage) map[string]string {
	var m map[string]string
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}
	return m
}

// DetectFromDocument returns producer detection from a parsed document.
func DetectFromDocument(doc *Document) lockfile.Detection {
	if doc == nil {
		return lockfile.Detection{}
	}
	switch doc.LockfileVersion {
	case 2:
		return lockfile.Detection{
			Format: FormatV2, ProducerMajor: 2, Confidence: lockfile.DetectionCertain,
			Evidence: []string{"lockfileVersion=2"},
		}
	case 3:
		return lockfile.Detection{
			Format: FormatV3, ProducerMajor: 3, Confidence: lockfile.DetectionCertain,
			Evidence: []string{"lockfileVersion=3"},
		}
	default:
		return lockfile.Detection{}
	}
}

// Encode serializes a document to JSON with deterministic ordering.
func Encode(doc *Document) ([]byte, error) {
	if doc == nil {
		return nil, apperr.New(apperr.Lockfile, "npm.encode", "document", "nil document")
	}
	payload := encodePayload(doc)
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(payload); err != nil {
		return nil, apperr.Wrap(apperr.Lockfile, "npm.encode", "package-lock.json", err)
	}
	return bytes.TrimSpace(buf.Bytes()), nil
}
