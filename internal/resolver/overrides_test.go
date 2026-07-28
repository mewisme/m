package resolver_test

import (
	"context"
	"testing"

	"github.com/mewisme/mew/internal/registry"
	"github.com/mewisme/mew/internal/resolver"
)

func overridePackuments() map[string]registry.Packument {
	return map[string]registry.Packument{
		"pkg-a": {
			Name:     "pkg-a",
			DistTags: map[string]string{"latest": "1.0.0"},
			Versions: map[string]registry.VersionMeta{
				"1.0.0": {
					Name:    "pkg-a",
					Version: "1.0.0",
					Dist:    registry.Dist{Integrity: "sha512-a", Tarball: "pkg-a-1.0.0.tgz"},
					Dependencies: map[string]string{
						"pkg-b": "^1.0.0",
					},
				},
			},
		},
		"pkg-b": {
			Name:     "pkg-b",
			DistTags: map[string]string{"latest": "1.2.0"},
			Versions: map[string]registry.VersionMeta{
				"1.0.0": {
					Name:    "pkg-b",
					Version: "1.0.0",
					Dist:    registry.Dist{Integrity: "sha512-b0", Tarball: "pkg-b-1.0.0.tgz"},
				},
				"1.2.0": {
					Name:    "pkg-b",
					Version: "1.2.0",
					Dist:    registry.Dist{Integrity: "sha512-b2", Tarball: "pkg-b-1.2.0.tgz"},
				},
			},
		},
	}
}

func TestResolveGlobalOverride(t *testing.T) {
	eng, _ := engineWithPackuments(t, overridePackuments())
	root := writeProject(t, `{
  "name": "root",
  "version": "1.0.0",
  "dependencies": { "pkg-a": "1.0.0" },
  "overrides": { "pkg-b": "1.0.0" }
}`)
	res, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range res.Graph.Packages {
		if p.ID.Name == "pkg-b" && p.ID.Version != "1.0.0" {
			t.Fatalf("override ignored: pkg-b@%s", p.ID.Version)
		}
	}
}

func TestResolveNestedOverride(t *testing.T) {
	eng, _ := engineWithPackuments(t, overridePackuments())
	root := writeProject(t, `{
  "name": "root",
  "version": "1.0.0",
  "dependencies": { "pkg-a": "1.0.0" },
  "overrides": { "pkg-a": { "pkg-b": "1.0.0" } }
}`)
	res, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range res.Graph.Packages {
		if p.ID.Name == "pkg-b" && p.ID.Version != "1.0.0" {
			t.Fatalf("nested override ignored: pkg-b@%s", p.ID.Version)
		}
	}
}

func TestResolveNpmAlias(t *testing.T) {
	packs := map[string]registry.Packument{
		"bar": {
			Name:     "bar",
			DistTags: map[string]string{"latest": "1.0.0"},
			Versions: map[string]registry.VersionMeta{
				"1.0.0": {
					Name:    "bar",
					Version: "1.0.0",
					Dist:    registry.Dist{Integrity: "sha512-bar", Tarball: "bar-1.0.0.tgz"},
				},
			},
		},
	}
	eng, _ := engineWithPackuments(t, packs)
	root := writeProject(t, `{
  "name": "root",
  "version": "1.0.0",
  "dependencies": { "foo": "npm:bar@^1.0.0" }
}`)
	res, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range res.Graph.Packages {
		if p.ID.Name == "bar" {
			found = true
		}
	}
	if !found {
		t.Fatal("alias target bar not resolved")
	}
}
