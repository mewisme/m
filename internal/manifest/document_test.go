package manifest_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/manifest"
	"github.com/mewisme/m/internal/testkit"
)

func TestParseRoundTripPreservesSource(t *testing.T) {
	root := testkit.ModuleRoot(t)
	path := filepath.Join(root, "fixtures", "projects", "manifest-format", "package.json")
	doc, err := manifest.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(doc.Source, want) {
		t.Fatalf("source mutated on load")
	}
	if doc.Name != "manifest-format" {
		t.Fatalf("name=%q", doc.Name)
	}
	if doc.Dependencies["@scope/pkg"] != "^1.0.0" {
		t.Fatalf("scoped dep=%v", doc.Dependencies)
	}
}

func TestDuplicateKeyRejected(t *testing.T) {
	_, err := manifest.Parse([]byte(`{"name":"a","name":"b"}`))
	if err == nil {
		t.Fatal("expected duplicate key error")
	}
	if apperr.CodeOf(err) != apperr.Manifest {
		t.Fatalf("code=%s", apperr.CodeOf(err))
	}
	if !strings.Contains(err.Error(), "duplicate key") {
		t.Fatalf("%v", err)
	}
}

func TestToNormalizedScoped(t *testing.T) {
	doc, err := manifest.Parse([]byte(`{
  "name": "x",
  "version": "1.0.0",
  "dependencies": {"@scope/pkg": "^1.0.0", "lodash": "4.17.21"},
  "devDependencies": {"typescript": "^5.0.0"}
}`))
	if err != nil {
		t.Fatal(err)
	}
	m, err := manifest.ToNormalized(doc)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Dependencies) != 3 {
		t.Fatalf("deps=%v", m.Dependencies)
	}
	found := false
	for _, d := range m.Dependencies {
		if d.Name == "@scope/pkg" && d.Kind == manifest.DepProd {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing scoped prod dep: %v", m.Dependencies)
	}
}

func TestSetFieldPreservesLayout(t *testing.T) {
	src := []byte("{\n    \"version\": \"0.0.1\",\n    \"name\": \"manifest-format\"\n}\n")
	doc, err := manifest.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.SetField("name", "renamed"); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(doc.Source, []byte(`"name": "renamed"`)) {
		t.Fatalf("source=%s", doc.Source)
	}
	if !bytes.Contains(doc.Source, []byte(`    "version"`)) {
		t.Fatalf("indent lost: %s", doc.Source)
	}
	if doc.Name != "renamed" {
		t.Fatalf("name=%q", doc.Name)
	}
}

func TestRemoveDependencyProdDev(t *testing.T) {
	doc, err := manifest.Parse([]byte(`{
  "name": "x",
  "version": "1.0.0",
  "dependencies": {"lodash": "4.17.21", "pkg-a": "^1.0.0"},
  "devDependencies": {"typescript": "^5.0.0", "eslint": "^8.0.0"}
}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.RemoveDependency("dependencies", "lodash"); err != nil {
		t.Fatal(err)
	}
	if _, ok := doc.Dependencies["lodash"]; ok {
		t.Fatalf("lodash still present: %v", doc.Dependencies)
	}
	if doc.Dependencies["pkg-a"] != "^1.0.0" {
		t.Fatalf("pkg-a=%q", doc.Dependencies["pkg-a"])
	}
	if !bytes.Contains(doc.Source, []byte(`"pkg-a"`)) {
		t.Fatalf("source missing pkg-a: %s", doc.Source)
	}
	if bytes.Contains(doc.Source, []byte(`"lodash"`)) {
		t.Fatalf("source still mentions lodash: %s", doc.Source)
	}
	if err := doc.RemoveDependency("devDependencies", "typescript"); err != nil {
		t.Fatal(err)
	}
	if _, ok := doc.DevDependencies["typescript"]; ok {
		t.Fatalf("typescript still present: %v", doc.DevDependencies)
	}
	if doc.DevDependencies["eslint"] != "^8.0.0" {
		t.Fatalf("eslint=%q", doc.DevDependencies["eslint"])
	}
	err = doc.RemoveDependency("dependencies", "missing")
	if err == nil {
		t.Fatal("expected not found")
	}
	if apperr.CodeOf(err) != apperr.NotFound {
		t.Fatalf("code=%s err=%v", apperr.CodeOf(err), err)
	}
}

func TestWriteAtomicGolden(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "package.json")
	before := []byte("{\n  \"name\": \"before\",\n  \"version\": \"1.0.0\"\n}\n")
	if err := os.WriteFile(path, before, 0o644); err != nil {
		t.Fatal(err)
	}
	doc, err := manifest.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.SetField("name", "after"); err != nil {
		t.Fatal(err)
	}
	if err := doc.Write(""); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	goldenDir := filepath.Join(testkit.ModuleRoot(t), "testdata", "manifest", "golden", "read-write")
	wantPath := filepath.Join(goldenDir, "after-set-name.json")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		_ = os.MkdirAll(goldenDir, 0o755)
		if err := os.WriteFile(wantPath, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestValidateNameVersionBin(t *testing.T) {
	if err := manifest.ValidateName("lodash"); err != nil {
		t.Fatal(err)
	}
	if err := manifest.ValidateName("@scope/pkg"); err != nil {
		t.Fatal(err)
	}
	if err := manifest.ValidateName("BadName"); err == nil {
		t.Fatal("expected uppercase reject")
	}
	if err := manifest.ValidateVersion("1.0.0"); err != nil {
		t.Fatal(err)
	}
	if err := manifest.ValidateVersion(" "); err == nil {
		t.Fatal("expected bad version")
	}
	if err := manifest.ValidateBin([]byte(`"./bin.js"`)); err != nil {
		t.Fatal(err)
	}
	if err := manifest.ValidateBin([]byte(`{"cli":"./cli.js"}`)); err != nil {
		t.Fatal(err)
	}
	if err := manifest.ValidateBin([]byte(`[]`)); err == nil {
		t.Fatal("expected bin reject")
	}
}

func TestLoadCachedInvalidate(t *testing.T) {
	manifest.ClearCacheForTest()
	dir := t.TempDir()
	path := filepath.Join(dir, "package.json")
	if err := os.WriteFile(path, []byte(`{"name":"c1","version":"1.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	d1, err := manifest.LoadCached(dir)
	if err != nil {
		t.Fatal(err)
	}
	if d1.Name != "c1" {
		t.Fatal(d1.Name)
	}
	manifest.Invalidate(dir)
	if err := os.WriteFile(path, []byte(`{"name":"c2","version":"1.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	// mtime may be same on coarse FS — Invalidate already dropped cache
	d2, err := manifest.LoadCached(dir)
	if err != nil {
		t.Fatal(err)
	}
	if d2.Name != "c2" {
		t.Fatalf("got %q", d2.Name)
	}
}
