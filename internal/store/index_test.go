package store

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/diagnostics"
	"github.com/mewisme/mew/internal/testkit"
)

type captureReporter struct {
	buf bytes.Buffer
}

func (c *captureReporter) Progress(ev diagnostics.Event) {
	c.buf.WriteString(ev.Phase)
}

func (c *captureReporter) Error(error) {}
func (c *captureReporter) Debug(msg string, attrs ...diagnostics.Attr) {
	c.buf.WriteString(msg)
	for _, a := range attrs {
		c.buf.WriteString(a.Key + "=" + a.Value)
	}
}
func (c *captureReporter) WorkspaceTask(diagnostics.WorkspaceTaskEvent) {}
func (c *captureReporter) ChildOutput(diagnostics.ChildOutputEvent, diagnostics.WorkspaceOutputMode) {
}
func (c *captureReporter) WorkspaceSummary(diagnostics.WorkspaceSummaryEvent) {}
func (c *captureReporter) EnvironmentPrepared(diagnostics.EnvironmentPreparedEvent) error {
	return nil
}
func (c *captureReporter) OperationStarted(diagnostics.OperationStartedEvent)     {}
func (c *captureReporter) OperationProgress(diagnostics.OperationProgressEvent)   {}
func (c *captureReporter) OperationCompleted(diagnostics.OperationCompletedEvent) {}
func (c *captureReporter) Notice(diagnostics.NoticeEvent)                         {}

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
	if !strings.Contains(rep.buf.String(), "warning: store index upsert failed") {
		t.Fatalf("expected index warning, got %q", rep.buf.String())
	}
}

func TestReconcileIndexMissingEntry(t *testing.T) {
	root := filepath.Join(t.TempDir(), "store")
	ps := NewPackageStore(root)
	tgz := filepath.Join(testkit.FixtureDir(t, "registry/v1"), "tarballs", "lodash-4.17.21.tgz")
	integrity := "sha256-758b80171fc185274170cb6db31a08042813d860a47b612d0671122a306b8b63"
	key, err := importIntegrity(context.Background(), ps, tgz, integrity)
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

func TestLoadIndexRejectsSchemaVersionZero(t *testing.T) {
	root := filepath.Join(t.TempDir(), "store")
	path := indexPathForTest(root)
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	raw := `{"schemaVersion":0,"packages":{}}` + "\n"
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadIndex(root); err == nil {
		t.Fatal("expected schemaVersion rejection")
	} else if !strings.Contains(err.Error(), "missing schemaVersion") {
		t.Fatalf("err=%v", err)
	}
}

func TestLoadIndexRejectsCorruptJSON(t *testing.T) {
	root := filepath.Join(t.TempDir(), "store")
	path := indexPathForTest(root)
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadIndex(root); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestStatusPartialIndexReconciles(t *testing.T) {
	root := filepath.Join(t.TempDir(), "store")
	ps := NewPackageStore(root)
	tgz := filepath.Join(testkit.FixtureDir(t, "registry/v1"), "tarballs", "lodash-4.17.21.tgz")
	integrity := "sha256-758b80171fc185274170cb6db31a08042813d860a47b612d0671122a306b8b63"
	key, err := importIntegrity(context.Background(), ps, tgz, integrity)
	if err != nil {
		t.Fatal(err)
	}
	idx, err := ReadIndex(root)
	if err != nil {
		t.Fatal(err)
	}
	delete(idx.Packages, key.String())
	raw, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(indexPathForTest(root), raw, 0o644); err != nil {
		t.Fatal(err)
	}

	count, _, err := ps.Status()
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("count=%d", count)
	}
	got, err := ReadIndex(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got.Packages[key.String()]; !ok {
		t.Fatal("expected reconciled index entry")
	}
}

func TestStatusFilesystemFallback(t *testing.T) {
	root := filepath.Join(t.TempDir(), "store")
	ps := NewPackageStore(root)
	tgz := filepath.Join(testkit.FixtureDir(t, "registry/v1"), "tarballs", "lodash-4.17.21.tgz")
	integrity := "sha256-758b80171fc185274170cb6db31a08042813d860a47b612d0671122a306b8b63"
	if _, err := importIntegrity(context.Background(), ps, tgz, integrity); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(indexPathForTest(root)); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	count, bytes, err := ps.Status()
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 || bytes <= 0 {
		t.Fatalf("count=%d bytes=%d", count, bytes)
	}
}

func TestReconcileIndexCorruptJSONRebuilds(t *testing.T) {
	root := filepath.Join(t.TempDir(), "store")
	ps := NewPackageStore(root)
	tgz := filepath.Join(testkit.FixtureDir(t, "registry/v1"), "tarballs", "lodash-4.17.21.tgz")
	integrity := "sha256-758b80171fc185274170cb6db31a08042813d860a47b612d0671122a306b8b63"
	key, err := importIntegrity(context.Background(), ps, tgz, integrity)
	if err != nil {
		t.Fatal(err)
	}
	path := indexPathForTest(root)
	if err := os.WriteFile(path, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := ps.ReconcileIndex()
	if err != nil {
		t.Fatal(err)
	}
	if result.Added != 1 {
		t.Fatalf("added=%d", result.Added)
	}
	matches, err := filepath.Glob(path + ".corrupt.*")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("quarantine files=%v", matches)
	}
	idx, err := ReadIndex(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := idx.Packages[key.String()]; !ok {
		t.Fatal("missing rebuilt entry")
	}
}

func TestStatusWrongKeySetSameCountRepairs(t *testing.T) {
	root := filepath.Join(t.TempDir(), "store")
	ps := NewPackageStore(root)
	tgz := filepath.Join(testkit.FixtureDir(t, "registry/v1"), "tarballs", "lodash-4.17.21.tgz")
	integrity := "sha256-758b80171fc185274170cb6db31a08042813d860a47b612d0671122a306b8b63"
	key, err := importIntegrity(context.Background(), ps, tgz, integrity)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := indexEntryFromPackage(ps.PackagePath(key))
	if err != nil {
		t.Fatal(err)
	}
	idx := &Index{
		SchemaVersion: indexSchemaVersion,
		Packages: map[string]IndexEntry{
			"sha256/deadbeef": {Integrity: "sha256-deadbeef", SizeBytes: entry.SizeBytes},
		},
	}
	raw, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(indexPathForTest(root), raw, 0o644); err != nil {
		t.Fatal(err)
	}

	count, _, err := ps.Status()
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("count=%d", count)
	}
	got, err := ReadIndex(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got.Packages[key.String()]; !ok {
		t.Fatal("expected index repaired to filesystem keys")
	}
	if _, ok := got.Packages["sha256/deadbeef"]; ok {
		t.Fatal("orphan index key should be removed")
	}
}
