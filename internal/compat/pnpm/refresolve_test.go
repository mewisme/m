package pnpm

import (
	"strings"
	"testing"
)

func TestResolveDependencyTargetGolden(t *testing.T) {
	idx := NewPackageIndex([]string{
		"b@2.0.0",
		"@scope/pkg@1.0.0",
		"react-dom@18.2.0#react@18.2.0",
	})
	cases := []struct {
		name, ref, want string
	}{
		{"b", "2.0.0", "b@2.0.0"},
		{"b", "b@2.0.0", "b@2.0.0"},
		{"@scope/pkg", "1.0.0", "@scope/pkg@1.0.0"},
		{"react-dom", "18.2.0(react@18.2.0)", "react-dom@18.2.0#react@18.2.0"},
		{"react-dom", "react-dom@18.2.0(react@18.2.0)", "react-dom@18.2.0#react@18.2.0"},
		{"local", "link:../pkg", "link:../pkg"},
	}
	for _, tc := range cases {
		target, err := ResolveDependencyTarget(tc.name, tc.ref, idx)
		if err != nil {
			t.Fatalf("%s %s: %v", tc.name, tc.ref, err)
		}
		if target.Key != tc.want {
			t.Fatalf("%s %s: got %q want %q", tc.name, tc.ref, target.Key, tc.want)
		}
	}
}

func TestResolveDependencyTargetDangling(t *testing.T) {
	idx := NewPackageIndex([]string{"a@1.0.0"})
	_, err := ResolveDependencyTarget("missing", "9.9.9", idx)
	if err == nil {
		t.Fatal("expected dangling error")
	}
	if !strings.Contains(err.Error(), "dangling") {
		t.Fatalf("expected dangling error: %v", err)
	}
}

func TestResolveDependencyTargetRejectsLooseMatch(t *testing.T) {
	idx := NewPackageIndex([]string{
		"pkg@1.0.0",
		"pkg@1.0.0-beta.1",
	})
	_, err := ResolveDependencyTarget("pkg", "0.0", idx)
	if err == nil {
		t.Fatal("expected dangling error for loose suffix match")
	}
	if !strings.Contains(err.Error(), "candidates") {
		t.Fatalf("expected candidates: %v", err)
	}
}

func TestResolveDependencyTargetPeerContext(t *testing.T) {
	idx := NewPackageIndex([]string{
		"acorn-jsx@5.3.2#acorn@8.18.0",
		"acorn@8.18.0",
	})
	target, err := ResolveDependencyTarget("acorn-jsx", "5.3.2(acorn@8.18.0)", idx)
	if err != nil {
		t.Fatal(err)
	}
	if target.Key != "acorn-jsx@5.3.2#acorn@8.18.0" {
		t.Fatalf("got %q", target.Key)
	}
}

func TestResolveDependencyTargetAlias(t *testing.T) {
	idx := NewPackageIndex([]string{"lodash@4.17.21"})
	target, err := ResolveDependencyTarget("alias-name", "npm:lodash@4.17.21", idx)
	if err == nil {
		// alias protocol may resolve directly as protocol ref
		if target.Key != "npm:lodash@4.17.21" {
			t.Fatalf("unexpected alias resolution: %q", target.Key)
		}
	}
}

func TestImporterDanglingRefFailsGraph(t *testing.T) {
	const src = `
lockfileVersion: '9.0'
importers:
  .:
    dependencies:
      ghost:
        specifier: ^9.9.9
        version: 9.9.9
packages:
  base@1.0.0:
    resolution: {integrity: sha512-base}
snapshots:
  base@1.0.0: {}
`
	doc, err := Decode([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	_, err = ToGraph(doc)
	if err == nil {
		t.Fatal("expected dangling importer ref error")
	}
}

func FuzzResolveDependencyTarget(f *testing.F) {
	f.Add("b", "2.0.0")
	f.Add("@scope/x", "1.0.0")
	f.Add("acorn-jsx", "5.3.2(acorn@8.18.0)")
	f.Fuzz(func(t *testing.T, name, ref string) {
		if len(name)+len(ref) > maxPackageKeyLen+maxPeerSuffixLen {
			return
		}
		idx := NewPackageIndex([]string{
			name + "@" + ref,
			ref,
			"acorn-jsx@5.3.2#acorn@8.18.0",
			"acorn@8.18.0",
		})
		_, _ = ResolveDependencyTarget(name, ref, idx)
	})
}
