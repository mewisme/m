package store

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/m/internal/diagnostics"
	"github.com/mewisme/m/internal/testkit"
)

type captureReporter struct {
	buf bytes.Buffer
}

func (c *captureReporter) Progress(diagnostics.Event) {}
func (c *captureReporter) Error(error)                {}
func (c *captureReporter) Debug(msg string, attrs ...diagnostics.Attr) {
	c.buf.WriteString(msg)
	for _, a := range attrs {
		c.buf.WriteString(a.Key + "=" + a.Value)
	}
}

func TestIndexUpsertFailureWarns(t *testing.T) {
	root := filepath.Join(t.TempDir(), "store")
	rep := &captureReporter{}
	ps := NewPackageStore(root)
	ps.Reporter = rep
	integrity := "sha256-758b80171fc185274170cb6db31a08042813d860a47b612d0671122a306b8b63"
	key, err := PackageKeyFromIntegrity(integrity)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	indexPath := filepath.Join(root, "index.json")
	if err := os.Mkdir(indexPath, 0o755); err != nil {
		t.Fatal(err)
	}

	ps.indexUpsertOrWarn(key, integrity, 123)
	if !strings.Contains(rep.buf.String(), "store index upsert failed") {
		t.Fatalf("expected index warning, got %q", rep.buf.String())
	}
}

func TestReconcileIndexMissingEntry(t *testing.T) {
	root := filepath.Join(t.TempDir(), "store")
	ps := NewPackageStore(root)
	tgz := filepath.Join(testkit.FixtureDir(t, "registry/v1"), "tarballs", "lodash-4.17.21.tgz")
	integrity := "sha256-758b80171fc185274170cb6db31a08042813d860a47b612d0671122a306b8b63"
	key, err := ps.ImportFromTarball(context.Background(), tgz, integrity)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(indexPathForTest(root)); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}

	result, err := ps.ReconcileIndex()
	if err != nil {
		t.Fatal(err)
	}
	if result.Added != 1 || result.Removed != 0 {
		t.Fatalf("added=%d removed=%d", result.Added, result.Removed)
	}
	idx, err := ReadIndex(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := idx.Packages[key.String()]; !ok {
		t.Fatal("missing reconciled entry")
	}
}

func TestReconcileIndexOrphanEntry(t *testing.T) {
	root := filepath.Join(t.TempDir(), "store")
	ps := NewPackageStore(root)
	idx := &Index{
		SchemaVersion: indexSchemaVersion,
		Packages: map[string]IndexEntry{
			"sha256/deadbeef": {Integrity: "sha256-deadbeef", SizeBytes: 1},
		},
	}
	raw, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	raw = append(raw, '\n')
	path := indexPathForTest(root)
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := ps.ReconcileIndex()
	if err != nil {
		t.Fatal(err)
	}
	if result.Removed != 1 {
		t.Fatalf("removed=%d", result.Removed)
	}
	got, err := ReadIndex(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Packages) != 0 {
		t.Fatalf("expected empty index, got %d", len(got.Packages))
	}
}

func indexPathForTest(root string) string {
	return filepath.Join(root, "index.json")
}
