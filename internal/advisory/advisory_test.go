package advisory

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/testkit"
)

func TestLoadAndMatchGraph(t *testing.T) {
	root := testkit.ModuleRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "testdata", "advisory", "fixture-osv.json"))
	if err != nil {
		t.Fatal(err)
	}
	db, err := Load(raw)
	if err != nil {
		t.Fatal(err)
	}
	if db.Digest == "" {
		t.Fatal("expected digest")
	}
	g := &graph.Graph{
		SchemaVersion: graph.SchemaVersion,
		Packages: []graph.Package{
			{ID: graph.PackageID{Name: "vuln-pkg", Version: "1.0.0"}},
			{ID: graph.PackageID{Name: "vuln-pkg", Version: "1.0.1"}},
			{ID: graph.PackageID{Name: "safe-pkg", Version: "1.0.0"}},
		},
	}
	report := db.MatchGraph(g)
	if len(report.Vulnerabilities) != 1 {
		t.Fatalf("vulns=%d want 1", len(report.Vulnerabilities))
	}
	v := report.Vulnerabilities[0]
	if v.ID != "CVE-2026-0001" {
		t.Fatalf("id=%q", v.ID)
	}
	if v.Package != "vuln-pkg" || v.Version != "1.0.0" {
		t.Fatalf("pkg=%s@%s", v.Package, v.Version)
	}
	if v.Severity != "critical" {
		t.Fatalf("severity=%q", v.Severity)
	}
}

func TestStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := Store{Dir: dir}
	root := testkit.ModuleRoot(t)
	if err := store.SeedFixture(root); err != nil {
		t.Fatal(err)
	}
	db, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(db.Entries) != 1 {
		t.Fatalf("entries=%d", len(db.Entries))
	}
}

func TestLoadSetsDigest(t *testing.T) {
	root := testkit.ModuleRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "testdata", "advisory", "fixture-osv.json"))
	if err != nil {
		t.Fatal(err)
	}
	want := Digest(raw)
	db, err := Load(raw)
	if err != nil {
		t.Fatal(err)
	}
	if db.Digest != want {
		t.Fatalf("digest=%q want %q", db.Digest, want)
	}
}
