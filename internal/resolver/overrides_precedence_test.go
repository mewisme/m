package resolver_test

import (
	"context"
	"testing"

	"github.com/mewisme/mew/internal/registry"
	"github.com/mewisme/mew/internal/resolver"
)

func precedencePackuments() map[string]registry.Packument {
	return map[string]registry.Packument{
		"root-pkg": {
			Name:     "root-pkg",
			DistTags: map[string]string{"latest": "1.0.0"},
			Versions: map[string]registry.VersionMeta{
				"1.0.0": {
					Name:    "root-pkg",
					Version: "1.0.0",
					Dist:    registry.Dist{Integrity: "sha512-root", Tarball: "root-pkg-1.0.0.tgz"},
					Dependencies: map[string]string{
						"nested-pkg": "1.0.0",
					},
				},
			},
		},
		"nested-pkg": {
			Name:     "nested-pkg",
			DistTags: map[string]string{"latest": "2.0.0"},
			Versions: map[string]registry.VersionMeta{
				"1.0.0": {
					Name:    "nested-pkg",
					Version: "1.0.0",
					Dist:    registry.Dist{Integrity: "sha512-n1", Tarball: "nested-pkg-1.0.0.tgz"},
					Dependencies: map[string]string{
						"leaf": "^1.0.0",
					},
				},
				"2.0.0": {
					Name:    "nested-pkg",
					Version: "2.0.0",
					Dist:    registry.Dist{Integrity: "sha512-n2", Tarball: "nested-pkg-2.0.0.tgz"},
				},
			},
		},
		"leaf": {
			Name:     "leaf",
			DistTags: map[string]string{"latest": "2.0.0"},
			Versions: map[string]registry.VersionMeta{
				"1.0.0": {
					Name:    "leaf",
					Version: "1.0.0",
					Dist:    registry.Dist{Integrity: "sha512-l1", Tarball: "leaf-1.0.0.tgz"},
				},
				"2.0.0": {
					Name:    "leaf",
					Version: "2.0.0",
					Dist:    registry.Dist{Integrity: "sha512-l2", Tarball: "leaf-2.0.0.tgz"},
				},
			},
		},
	}
}

func TestOverrideLongestSpecificWins(t *testing.T) {
	eng, _ := engineWithPackuments(t, precedencePackuments())
	root := writeProject(t, `{
  "name": "root",
  "version": "1.0.0",
  "dependencies": { "root-pkg": "1.0.0" },
  "overrides": {
    "leaf": "2.0.0",
    "root-pkg": { "nested-pkg": { "leaf": "1.0.0" } }
  }
}`)
	res, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range res.Graph.Packages {
		if p.ID.Name == "leaf" && p.ID.Version != "1.0.0" {
			t.Fatalf("nested override should win: leaf@%s", p.ID.Version)
		}
	}
	for _, d := range res.Decisions {
		if d.Package == "leaf" && d.OverrideFrom == "" {
			t.Fatalf("expected override decision for leaf: %#v", d)
		}
	}
}
