package pnpm_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/mewisme/mew/internal/compat/pnpm"
	"github.com/mewisme/mew/internal/lockfile"
)

func TestImporterSectionRoundTrip(t *testing.T) {
	const src = `
lockfileVersion: '9.0'
importers:
  .:
    dependencies:
      prod:
        specifier: ^1.0.0
        version: 1.0.0
    devDependencies:
      dev:
        specifier: ^2.0.0
        version: 2.0.0
    optionalDependencies:
      opt:
        specifier: ^3.0.0
        version: 3.0.0
    dependenciesMeta:
      prod:
        injected: true
    publishDirectory: dist
    customFlag: true
packages:
  prod@1.0.0:
    resolution: {integrity: sha512-p}
  dev@2.0.0:
    resolution: {integrity: sha512-d}
  opt@3.0.0:
    resolution: {integrity: sha512-o}
snapshots:
  prod@1.0.0: {}
  dev@2.0.0: {}
  opt@3.0.0: {}
`
	doc, err := pnpm.Decode([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if doc.Importers["."].PublishDirectory != "dist" {
		t.Fatalf("publishDirectory=%q", doc.Importers["."].PublishDirectory)
	}
	meta := doc.Importers["."].DependenciesMeta["prod"].(map[string]any)
	if meta["injected"] != true {
		t.Fatalf("dependenciesMeta: %#v", doc.Importers["."].DependenciesMeta)
	}
	if _, ok := doc.Importers["."].Extra["customFlag"]; !ok {
		t.Fatal("expected custom importer key in Extra")
	}
	out, err := pnpm.Encode(doc)
	if err != nil {
		t.Fatal(err)
	}
	doc2, err := pnpm.Decode(out)
	if err != nil {
		t.Fatal(err)
	}
	if doc2.Importers["."].PublishDirectory != "dist" {
		t.Fatal("publishDirectory not preserved")
	}
	if !bytes.Equal(doc.Importers["."].Extra["customFlag"], doc2.Importers["."].Extra["customFlag"]) {
		t.Fatal("custom importer key not preserved")
	}
}

func TestOptionalNotPromotedToProd(t *testing.T) {
	const src = `
lockfileVersion: '9.0'
importers:
  .:
    optionalDependencies:
      opt:
        specifier: ^1.0.0
        version: 1.0.0
packages:
  opt@1.0.0:
    resolution: {integrity: sha512-o}
snapshots:
  opt@1.0.0: {}
`
	doc, err := pnpm.Decode([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	g, err := pnpm.ToGraph(doc)
	if err != nil {
		t.Fatal(err)
	}
	out, err := pnpm.FromGraph(g, doc, lockfile.Detection{Format: pnpm.FormatV9, ProducerMajor: 9, ExplicitMajor: true}, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Importers["."].Dependencies) != 0 {
		t.Fatal("optional dep promoted to prod")
	}
	if len(out.Importers["."].OptionalDependencies) != 1 {
		t.Fatal("optional dep missing after fromGraph")
	}
}

func TestImporterExtraSurvivesMutation(t *testing.T) {
	doc := &pnpm.Document{
		LockfileVersion: "9.0",
		Importers: map[string]pnpm.ImporterSection{
			".": {
				Dependencies: map[string]pnpm.ImporterDep{
					"a": {Specifier: "^1.0.0", Version: "1.0.0"},
				},
				Extra: map[string]json.RawMessage{
					"customFlag": json.RawMessage(`true`),
				},
			},
		},
		Packages: map[string]pnpm.PackageEntry{
			"a@1.0.0": {Resolution: map[string]any{"integrity": "sha512-a"}},
		},
		Snapshots: map[string]map[string]any{
			"a@1.0.0": {},
		},
	}
	g, err := pnpm.ToGraph(doc)
	if err != nil {
		t.Fatal(err)
	}
	out, err := pnpm.FromGraph(g, doc, lockfile.Detection{Format: pnpm.FormatV9, ProducerMajor: 9, ExplicitMajor: true}, "")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(doc.Importers["."].Extra["customFlag"], out.Importers["."].Extra["customFlag"]) {
		t.Fatal("importer extra lost in fromGraph")
	}
}
