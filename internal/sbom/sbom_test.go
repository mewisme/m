package sbom_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/lockfile/mlock"
	"github.com/mewisme/mew/internal/sbom"
	"github.com/mewisme/mew/internal/testkit"
)

func mediumGraphFromLock(t *testing.T) *graph.Graph {
	t.Helper()
	moduleRoot := testkit.ModuleRoot(t)
	lockPath := filepath.Join(moduleRoot, ".cache", "mew", "bench", "medium-graph", "project", "m.lock")
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Skipf("medium-graph lock unavailable: %v", err)
	}
	doc, err := mlock.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	g, err := mlock.ToGraph(doc)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func fixedSBOMOpts() sbom.SBOMOptions {
	return sbom.SBOMOptions{
		ProjectName: "medium-graph",
		GeneratedAt: time.Date(2026, 7, 30, 0, 0, 0, 0, time.UTC),
	}
}

func TestExportCycloneDXMediumGraphGolden(t *testing.T) {
	g := mediumGraphFromLock(t)
	out, err := sbom.ExportCycloneDX(g, fixedSBOMOpts())
	if err != nil {
		t.Fatal(err)
	}
	goldenPath := filepath.Join(testkit.ModuleRoot(t), "fixtures", "sbom", "medium-graph-cyclonedx-golden.json")
	golden, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if string(out) != string(golden) {
		t.Fatalf("cyclonedx output differs from golden\n--- got\n%s\n--- want\n%s", out, golden)
	}
}

func TestExportCycloneDXIncludesTransitivePackages(t *testing.T) {
	g := mediumGraphFromLock(t)
	out, err := sbom.ExportCycloneDX(g, fixedSBOMOpts())
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	for _, pkg := range []string{"pkg-a", "pkg-b", "pkg-c", "lodash", "pkg-cli"} {
		if !strings.Contains(text, pkg) {
			t.Fatalf("missing %s", pkg)
		}
	}
}

func TestExportCycloneDXRedactInternal(t *testing.T) {
	g := mediumGraphFromLock(t)
	opts := fixedSBOMOpts()
	opts.RedactInternal = true
	out, err := sbom.ExportCycloneDX(g, opts)
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	if strings.Contains(text, "@scope/pkg") || strings.Contains(text, "%40scope") {
		t.Fatalf("scope package not redacted: %s", text)
	}
	if !strings.Contains(text, "[redacted]") {
		t.Fatal("expected redacted placeholder")
	}
}

func TestExportSPDXSmoke(t *testing.T) {
	g := mediumGraphFromLock(t)
	out, err := sbom.ExportSPDX(g, fixedSBOMOpts())
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	if !strings.HasPrefix(text, "SPDXVersion: SPDX-2.3\n") {
		t.Fatalf("bad header: %q", text[:min(40, len(text))])
	}
	for _, want := range []string{"PackageName: lodash", "Checksum: SHA256:", "ExternalRef: PACKAGE-MANAGER purl pkg:npm/lodash@4.17.21"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in:\n%s", want, text)
		}
	}
}

func TestExportInvalidRedactPattern(t *testing.T) {
	g, err := graph.NewBuilder().
		Importer(graph.RootImporter, "root").
		Package(graph.PackageID{Name: "pkg-a", Version: "1.0.0"}, "", "").
		Build()
	if err != nil {
		t.Fatal(err)
	}
	_, err = sbom.ExportCycloneDX(g, sbom.SBOMOptions{RedactPattern: "["})
	if err == nil {
		t.Fatal("expected invalid pattern error")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
