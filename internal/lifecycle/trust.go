package lifecycle

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
)

const trustSchemaVersion = 1

// TrustStore is the project-local trusted package allowlist.
type TrustStore struct {
	SchemaVersion int      `json:"schemaVersion"`
	Packages      []string `json:"packages"`
	path          string
	set           map[string]struct{}
}

// TrustFilePath returns <project>/.mew/trusted-packages.json.
func TrustFilePath(projectRoot string) string {
	return filepath.Join(projectRoot, ".mew", "trusted-packages.json")
}

// LoadTrust reads the trusted package list, returning an empty store when missing.
func LoadTrust(projectRoot string) (*TrustStore, error) {
	path := TrustFilePath(projectRoot)
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &TrustStore{SchemaVersion: trustSchemaVersion, path: path, set: map[string]struct{}{}}, nil
		}
		return nil, apperr.Wrap(apperr.IO, "lifecycle.trust", path, err)
	}
	var doc TrustStore
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, apperr.Wrap(apperr.IO, "lifecycle.trust", path, err)
	}
	if doc.SchemaVersion == 0 {
		doc.SchemaVersion = trustSchemaVersion
	}
	if doc.SchemaVersion != trustSchemaVersion {
		return nil, apperr.New(apperr.Config, "lifecycle.trust", path, "unsupported schemaVersion")
	}
	doc.path = path
	doc.rebuildSet()
	sort.Strings(doc.Packages)
	return &doc, nil
}

func (t *TrustStore) rebuildSet() {
	if t.set == nil {
		t.set = map[string]struct{}{}
	}
	for k := range t.set {
		delete(t.set, k)
	}
	for _, p := range t.Packages {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		t.set[p] = struct{}{}
	}
}

// IsTrusted reports whether pkg is on the allowlist.
func (t *TrustStore) IsTrusted(pkg string) bool {
	if t == nil {
		return false
	}
	_, ok := t.set[strings.TrimSpace(pkg)]
	return ok
}

// AddTrusted appends pkg when not already trusted and persists the store.
func (t *TrustStore) AddTrusted(pkg string) error {
	if t == nil {
		return apperr.New(apperr.Internal, "lifecycle.trust", pkg, "nil trust store")
	}
	pkg = strings.TrimSpace(pkg)
	if pkg == "" {
		return apperr.New(apperr.Usage, "lifecycle.trust", pkg, "empty package name")
	}
	if t.IsTrusted(pkg) {
		return nil
	}
	t.Packages = append(t.Packages, pkg)
	sort.Strings(t.Packages)
	t.rebuildSet()
	return t.Save()
}

// Save writes the trust store to disk.
func (t *TrustStore) Save() error {
	if t == nil {
		return apperr.New(apperr.Internal, "lifecycle.trust", "", "nil trust store")
	}
	if t.path == "" {
		return apperr.New(apperr.Internal, "lifecycle.trust", "", "missing trust path")
	}
	if err := os.MkdirAll(filepath.Dir(t.path), 0o755); err != nil {
		return apperr.Wrap(apperr.IO, "lifecycle.trust", t.path, err)
	}
	doc := TrustStore{SchemaVersion: trustSchemaVersion, Packages: append([]string(nil), t.Packages...)}
	sort.Strings(doc.Packages)
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return apperr.Wrap(apperr.IO, "lifecycle.trust", t.path, err)
	}
	if err := os.WriteFile(t.path, buf.Bytes(), 0o644); err != nil {
		return apperr.Wrap(apperr.IO, "lifecycle.trust", t.path, err)
	}
	return nil
}
