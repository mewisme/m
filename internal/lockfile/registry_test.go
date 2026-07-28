package lockfile_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	_ "github.com/mewisme/mew/internal/compat/nub"
	_ "github.com/mewisme/mew/internal/compat/pnpm"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/lockfile"
	_ "github.com/mewisme/mew/internal/lockfile/mlock"
	"github.com/mewisme/mew/internal/project"
)

func TestAdapterForMew(t *testing.T) {
	if lockfile.AdapterFor(project.IdentityMew) == nil {
		t.Fatal("expected mew adapter")
	}
	if lockfile.AdapterFor(project.IdentityPNPM) == nil {
		t.Fatal("expected pnpm adapter")
	}
}

func TestExtAdapterForMew(t *testing.T) {
	ext, ok := lockfile.ExtAdapterFor(project.IdentityMew)
	if !ok || ext == nil {
		t.Fatal("expected mew ext adapter")
	}
	nubExt, ok := lockfile.ExtAdapterFor(project.IdentityNub)
	if !ok || nubExt == nil {
		t.Fatal("expected nub ext adapter")
	}
}

func TestDetectPnpmFixtures(t *testing.T) {
	root := filepath.Join("..", "..", "fixtures", "locks", "pnpm")
	cases := []struct {
		dir    string
		format string
		major  int
		conf   lockfile.DetectionConfidence
	}{
		{"v9", "pnpm-v9", 0, lockfile.DetectionInferred},
		{"v10", "pnpm-v10", 10, lockfile.DetectionInferred},
		{"v11", "pnpm-v11", 11, lockfile.DetectionInferred},
	}
	for _, tc := range cases {
		data, err := os.ReadFile(filepath.Join(root, tc.dir, "pnpm-lock.yaml"))
		if err != nil {
			t.Fatalf("%s: %v", tc.dir, err)
		}
		det, err := lockfile.DetectPnpm(data)
		if err != nil {
			t.Fatalf("%s: %v", tc.dir, err)
		}
		if det.Format != tc.format {
			t.Fatalf("%s: format=%s want %s", tc.dir, det.Format, tc.format)
		}
		if det.ProducerMajor != tc.major {
			t.Fatalf("%s: major=%d want %d", tc.dir, det.ProducerMajor, tc.major)
		}
		if det.Confidence != tc.conf {
			t.Fatalf("%s: confidence=%s want %s", tc.dir, det.Confidence, tc.conf)
		}
	}
}

func TestDetectPnpmRejectsUnsupportedV6Fixture(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "fixtures", "locks", "pnpm", "unsupported", "v6", "pnpm-lock.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = lockfile.DetectPnpm(data)
	if err == nil {
		t.Fatal("expected legacy rejection")
	}
	if apperr.CodeOf(err) != apperr.LockUnsupported {
		t.Fatalf("code=%s", apperr.CodeOf(err))
	}
}

func TestDetectPnpmAmbiguousNotCertified(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "fixtures", "locks", "pnpm", "v9", "pnpm-lock.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	det, err := lockfile.DetectPnpm(data)
	if err != nil {
		t.Fatal(err)
	}
	if det.Certified() {
		t.Fatal("ambiguous v9-shaped lock must not be certified")
	}
}

func TestWritePreservingUnchangedGraph(t *testing.T) {
	ext, ok := lockfile.ExtAdapterFor(project.IdentityMew)
	if !ok {
		t.Fatal("missing ext adapter")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "m.lock")
	g := &graph.Graph{
		SchemaVersion: graph.SchemaVersion,
		Importers:     []graph.Importer{{ID: graph.RootImporter}},
	}
	if err := ext.Write(context.Background(), path, g); err != nil {
		t.Fatal(err)
	}
	prior, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := ext.WritePreserving(context.Background(), path, g, prior, nil, lockfile.Detection{Confidence: lockfile.DetectionCertain}); err != nil {
		t.Fatal(err)
	}
}

func TestWritePreservingAmbiguousDetection(t *testing.T) {
	ext, ok := lockfile.ExtAdapterFor(project.IdentityMew)
	if !ok {
		t.Fatal("missing ext adapter")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "m.lock")
	base := &graph.Graph{
		SchemaVersion: graph.SchemaVersion,
		Importers:     []graph.Importer{{ID: graph.RootImporter}},
	}
	if err := ext.Write(context.Background(), path, base); err != nil {
		t.Fatal(err)
	}
	prior, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	g := &graph.Graph{
		SchemaVersion: graph.SchemaVersion,
		Importers:     []graph.Importer{{ID: graph.RootImporter}},
		Packages:      []graph.Package{{ID: graph.PackageID{Name: "left-pad", Version: "1.3.0"}}},
		Edges:         []graph.Edge{{From: ".", Name: "left-pad", To: "left-pad@1.3.0", Kind: graph.DepProd, Range: "1.3.0"}},
	}
	err = ext.WritePreserving(context.Background(), path, g, prior, nil, lockfile.Detection{Confidence: lockfile.DetectionInferred})
	if err == nil {
		t.Fatal("expected ambiguous write failure")
	}
	if apperr.CodeOf(err) != apperr.LockAmbiguous {
		t.Fatalf("code=%s want %s", apperr.CodeOf(err), apperr.LockAmbiguous)
	}
}
