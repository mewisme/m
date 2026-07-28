package manifest

import (
	"encoding/json"

	"github.com/mewisme/mew/internal/apperr"
)

// Catalog is a map of catalog entry names to version ranges.
type Catalog map[string]string

// ParseCatalog unmarshals the package.json catalog field.
func ParseCatalog(raw json.RawMessage) (Catalog, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var m map[string]string
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, apperr.Wrap(apperr.Manifest, "manifest.catalog", "catalog", err)
	}
	if len(m) == 0 {
		return Catalog{}, nil
	}
	out := make(Catalog, len(m))
	for k, v := range m {
		k = trimSpace(k)
		v = trimSpace(v)
		if k == "" {
			return nil, apperr.New(apperr.Manifest, "manifest.catalog", "catalog", "empty catalog entry name")
		}
		if v == "" {
			return nil, apperr.New(apperr.Manifest, "manifest.catalog", k, "empty catalog range")
		}
		out[k] = v
	}
	return out, nil
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}

// ResolveCatalogEntry looks up a catalog entry by key in the default catalog.
func (c Catalog) ResolveEntry(entryKey string) (string, error) {
	if len(c) == 0 {
		return "", apperr.New(apperr.Manifest, "manifest.catalog", entryKey, "catalog not defined")
	}
	v, ok := c[entryKey]
	if !ok {
		return "", apperr.New(apperr.Manifest, "manifest.catalog", entryKey, "undefined catalog entry")
	}
	return v, nil
}
