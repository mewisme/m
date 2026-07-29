package bun

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/lockfile"
)

var bunLockbMagic = []byte("#!/usr/bin/env bun")

var knownTopLevel = map[string]struct{}{
	"lockfileVersion":     {},
	"configVersion":       {},
	"workspaces":          {},
	"packages":            {},
	"patchedDependencies": {},
	"overrides":           {},
	"catalog":             {},
	"catalogs":            {},
	"trustedDependencies": {},
}

// Decode parses text bun.lock; rejects binary bun.lockb.
func Decode(data []byte) (*Document, error) {
	if err := validateLockInput(data); err != nil {
		return nil, err
	}
	if isBinaryLockb(data) {
		return nil, lockfile.NewUnsupported("bun.decode", "bun.lockb",
			"binary bun.lockb is not supported; convert to text bun.lock with: bun install --save-text-lockfile --frozen-lockfile --lockfile-only")
	}
	stripped := stripJSONC(data)
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(stripped, &raw); err != nil {
		return nil, apperr.Wrap(apperr.Lockfile, "bun.decode", "bun.lock", err)
	}
	if len(raw) > maxMapEntries {
		return nil, apperr.New(apperr.Lockfile, "bun.decode", "bun.lock",
			fmt.Sprintf("exceeds %d top-level keys", maxMapEntries))
	}

	doc := &Document{
		Workspaces: map[string]WorkspaceEntry{},
		Packages:   map[string]PackageArray{},
		Extensions: lockfile.Extensions{},
	}

	for k, v := range raw {
		if _, known := knownTopLevel[k]; known {
			continue
		}
		doc.Extensions[k] = append(json.RawMessage(nil), v...)
	}

	if verRaw, ok := raw["lockfileVersion"]; ok {
		var ver int
		if err := json.Unmarshal(verRaw, &ver); err != nil {
			return nil, apperr.Wrap(apperr.Lockfile, "bun.decode", "lockfileVersion", err)
		}
		doc.LockfileVersion = ver
	}
	if cfgRaw, ok := raw["configVersion"]; ok {
		var ver int
		_ = json.Unmarshal(cfgRaw, &ver)
		doc.ConfigVersion = ver
	}
	if wsRaw, ok := raw["workspaces"]; ok {
		var wsMap map[string]json.RawMessage
		if err := json.Unmarshal(wsRaw, &wsMap); err != nil {
			return nil, apperr.Wrap(apperr.Lockfile, "bun.decode", "workspaces", err)
		}
		for path, entryRaw := range wsMap {
			entry, err := decodeWorkspaceEntry(entryRaw)
			if err != nil {
				return nil, apperr.Wrap(apperr.Lockfile, "bun.decode", "workspaces."+path, err)
			}
			doc.Workspaces[path] = entry
		}
	}
	if pkgRaw, ok := raw["packages"]; ok {
		var pkgMap map[string]json.RawMessage
		if err := json.Unmarshal(pkgRaw, &pkgMap); err != nil {
			return nil, apperr.Wrap(apperr.Lockfile, "bun.decode", "packages", err)
		}
		if len(pkgMap) > maxMapEntries {
			return nil, apperr.New(apperr.Lockfile, "bun.decode", "packages",
				fmt.Sprintf("exceeds %d package entries", maxMapEntries))
		}
		for name, arrRaw := range pkgMap {
			if err := validatePackageName(name); err != nil {
				return nil, err
			}
			var arr []json.RawMessage
			if err := json.Unmarshal(arrRaw, &arr); err != nil {
				return nil, apperr.Wrap(apperr.Lockfile, "bun.decode", "packages."+name, err)
			}
			doc.Packages[name] = PackageArray(arr)
		}
	}
	return doc, nil
}

func decodeWorkspaceEntry(raw json.RawMessage) (WorkspaceEntry, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return WorkspaceEntry{}, err
	}
	entry := WorkspaceEntry{Extra: map[string]json.RawMessage{}}
	for k, v := range obj {
		switch k {
		case "name":
			_ = json.Unmarshal(v, &entry.Name)
		case "dependencies":
			_ = json.Unmarshal(v, &entry.Dependencies)
		case "devDependencies":
			_ = json.Unmarshal(v, &entry.DevDependencies)
		case "optionalDependencies":
			_ = json.Unmarshal(v, &entry.OptionalDependencies)
		default:
			entry.Extra[k] = append(json.RawMessage(nil), v...)
		}
	}
	return entry, nil
}

func isBinaryLockb(data []byte) bool {
	trimmed := bytes.TrimLeftFunc(data, unicode.IsSpace)
	if len(trimmed) == 0 {
		return false
	}
	if trimmed[0] == '{' {
		return false
	}
	if bytes.HasPrefix(trimmed, bunLockbMagic) {
		return true
	}
	// ponytail: non-JSON leading byte heuristic; upgrade path is explicit magic table.
	return trimmed[0] != '{' && trimmed[0] != '['
}

func stripJSONC(b []byte) []byte {
	var out bytes.Buffer
	inStr := false
	esc := false
	for i := 0; i < len(b); i++ {
		c := b[i]
		if inStr {
			out.WriteByte(c)
			if esc {
				esc = false
				continue
			}
			if c == '\\' {
				esc = true
				continue
			}
			if c == '"' {
				inStr = false
			}
			continue
		}
		if c == '"' {
			inStr = true
			out.WriteByte(c)
			continue
		}
		if c == '/' && i+1 < len(b) {
			if b[i+1] == '/' {
				i += 2
				for i < len(b) && b[i] != '\n' {
					i++
				}
				if i < len(b) {
					out.WriteByte('\n')
				}
				continue
			}
			if b[i+1] == '*' {
				i += 2
				for i+1 < len(b) && (b[i] != '*' || b[i+1] != '/') {
					i++
				}
				i++
				continue
			}
		}
		if c == ',' {
			j := i + 1
			for j < len(b) && (b[j] == ' ' || b[j] == '\t' || b[j] == '\n' || b[j] == '\r') {
				j++
			}
			if j < len(b) && (b[j] == '}' || b[j] == ']') {
				continue
			}
		}
		out.WriteByte(c)
	}
	return out.Bytes()
}

// ParseResolution splits a bun resolution string into name and version.
func ParseResolution(res string) (name, version string, err error) {
	res = strings.TrimSpace(res)
	if res == "" {
		return "", "", apperr.New(apperr.Lockfile, "bun.identity", res, "empty resolution")
	}
	if strings.Contains(res, "://") || strings.HasPrefix(res, "github:") || strings.HasPrefix(res, "git+") {
		return res, "", nil
	}
	at := strings.LastIndexByte(res, '@')
	if at <= 0 || at == len(res)-1 {
		return "", "", apperr.New(apperr.Lockfile, "bun.identity", res, "malformed resolution")
	}
	return res[:at], res[at+1:], nil
}

func parsePackageTuple(arr PackageArray) (resolution, registry, integrity string, info PackageInfo, err error) {
	if len(arr) == 0 {
		return "", "", "", PackageInfo{}, apperr.New(apperr.Lockfile, "bun.decode", "package", "empty package tuple")
	}
	_ = json.Unmarshal(arr[0], &resolution)
	if len(arr) > 1 {
		_ = json.Unmarshal(arr[1], &registry)
	}
	if len(arr) > 2 {
		_ = json.Unmarshal(arr[2], &info)
	}
	if len(arr) > 3 {
		_ = json.Unmarshal(arr[3], &integrity)
	}
	return resolution, registry, integrity, info, nil
}
