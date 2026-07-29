package pnpm

import "testing"

func TestParseAliasFromImporterDepPeerContext(t *testing.T) {
	actual, ref, ok := ParseAliasFromImporterDep("my-acorn-jsx", "npm:acorn-jsx@5.3.2", "acorn-jsx@5.3.2(acorn@8.18.0)")
	if !ok {
		t.Fatal("expected alias")
	}
	if actual != "acorn-jsx" || ref != "acorn-jsx@5.3.2(acorn@8.18.0)" {
		t.Fatalf("got actual=%q ref=%q", actual, ref)
	}
}

func TestParseAliasFromImporterDepPeerContextVersionOnly(t *testing.T) {
	_, _, ok := ParseAliasFromImporterDep("my-acorn-jsx", "", "acorn-jsx@5.3.2(acorn@8.18.0)")
	if ok {
		t.Fatal("version-only peer-context ref is not an alias without npm: specifier")
	}
}
