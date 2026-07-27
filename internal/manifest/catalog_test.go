package manifest_test

import (
	"testing"

	"github.com/mewisme/m/internal/manifest"
)

func TestParseCatalogSpecifier(t *testing.T) {
	sp, err := manifest.ParseSpecifier("react", "catalog:")
	if err != nil {
		t.Fatal(err)
	}
	if sp.Protocol != manifest.ProtocolCatalog || sp.Range != "react" {
		t.Fatalf("got %#v", sp)
	}
	sp, err = manifest.ParseSpecifier("react", "catalog:default")
	if err != nil {
		t.Fatal(err)
	}
	if sp.Protocol != manifest.ProtocolCatalog || sp.Range != "react" {
		t.Fatalf("got %#v", sp)
	}
	sp, err = manifest.ParseSpecifier("react", "catalog:lodash")
	if err != nil {
		t.Fatal(err)
	}
	if sp.Range != "lodash" {
		t.Fatalf("range=%q", sp.Range)
	}
}

func TestParseCatalogResolveEntry(t *testing.T) {
	cat, err := manifest.ParseCatalog([]byte(`{"react":"^18.0.0"}`))
	if err != nil {
		t.Fatal(err)
	}
	v, err := cat.ResolveEntry("react")
	if err != nil || v != "^18.0.0" {
		t.Fatalf("v=%q err=%v", v, err)
	}
	_, err = cat.ResolveEntry("missing")
	if err == nil {
		t.Fatal("expected error")
	}
}
