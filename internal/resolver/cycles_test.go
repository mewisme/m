package resolver_test

import (
	"context"
	"testing"

	"github.com/mewisme/m/internal/registry"
	"github.com/mewisme/m/internal/resolver"
)

func cyclePackuments() map[string]registry.Packument {
	return map[string]registry.Packument{
		"parent": {
			Name:     "parent",
			DistTags: map[string]string{"latest": "1.0.0"},
			Versions: map[string]registry.VersionMeta{
				"1.0.0": {
					Name:    "parent",
					Version: "1.0.0",
					Dist:    registry.Dist{Integrity: "sha512-parent", Tarball: "parent-1.0.0.tgz"},
					Dependencies: map[string]string{
						"child": "1.0.0",
					},
				},
				"2.0.0": {
					Name:    "parent",
					Version: "2.0.0",
					Dist:    registry.Dist{Integrity: "sha512-parent2", Tarball: "parent-2.0.0.tgz"},
					Dependencies: map[string]string{
						"child": "2.0.0",
					},
				},
			},
		},
		"child": {
			Name:     "child",
			DistTags: map[string]string{"latest": "2.0.0"},
			Versions: map[string]registry.VersionMeta{
				"1.0.0": {
					Name:    "child",
					Version: "1.0.0",
					Dist:    registry.Dist{Integrity: "sha512-child1", Tarball: "child-1.0.0.tgz"},
					Dependencies: map[string]string{
						"parent": "1.0.0",
					},
				},
				"2.0.0": {
					Name:    "child",
					Version: "2.0.0",
					Dist:    registry.Dist{Integrity: "sha512-child2", Tarball: "child-2.0.0.tgz"},
					Dependencies: map[string]string{
						"parent": "2.0.0",
					},
				},
			},
		},
	}
}

func TestResolveDifferentVersionCycle(t *testing.T) {
	eng, _ := engineWithPackuments(t, cyclePackuments())
	root := writeProject(t, `{
  "name": "root",
  "version": "1.0.0",
  "dependencies": { "parent": "1.0.0" }
}`)
	res, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{})
	if err != nil {
		t.Fatalf("different-version cycle should resolve: %v", err)
	}
	versions := map[string]string{}
	for _, p := range res.Graph.Packages {
		versions[p.ID.Name] = p.ID.Version
	}
	if versions["parent"] != "1.0.0" || versions["child"] != "1.0.0" {
		t.Fatalf("versions=%v", versions)
	}
}
