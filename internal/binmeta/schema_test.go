package binmeta

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/linker"
)

func TestSchemaRoundTrip(t *testing.T) {
	dir := t.TempDir()
	nm := filepath.Join(dir, "node_modules")
	doc, err := BuildDocument(PublishInput{
		NodeModules:      nm,
		ImporterIdentity: ".",
		GenerationID:     "gen-1",
		LayoutMode:       LayoutHoisted,
		Sources: []linker.BinSource{{
			Cmd: "eslint", Target: "bin/eslint.js", PackageDir: filepath.Join(nm, "eslint"),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := Write(nm, doc); err != nil {
		t.Fatal(err)
	}
	got, err := Read(nm)
	if err != nil {
		t.Fatal(err)
	}
	if got.GenerationID != "gen-1" || len(got.Records) != 1 {
		t.Fatalf("got=%+v", got)
	}
}

func TestStaleGeneration(t *testing.T) {
	doc := &Document{SchemaVersion: SchemaVersion, GenerationID: "a", Fingerprint: "fp"}
	if !Stale(doc, "b", "fp") {
		t.Fatal("expected stale")
	}
	if !GenerationMatches(doc, "a", "fp") {
		t.Fatal("expected match")
	}
}

func TestValidateRejectsDuplicate(t *testing.T) {
	doc := &Document{
		SchemaVersion: SchemaVersion,
		GenerationID:  "g",
		Fingerprint:   "f",
		Records: []Record{
			{DeclaredBin: "tsc", MaterializedShim: "a", DependencyName: "a"},
			{DeclaredBin: "tsc", MaterializedShim: "b", DependencyName: "b"},
		},
	}
	if err := Validate(doc); err == nil {
		t.Fatal("expected duplicate error")
	}
}

func TestPublishCreatesFile(t *testing.T) {
	dir := t.TempDir()
	nm := filepath.Join(dir, "node_modules")
	if err := os.MkdirAll(nm, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Publish(PublishInput{
		NodeModules: nm, GenerationID: "g1", ImporterIdentity: ".",
		Sources: []linker.BinSource{{Cmd: "echo", Target: "cli.js", PackageDir: filepath.Join(nm, "echo")}},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(Path(nm)); err != nil {
		t.Fatal(err)
	}
}
