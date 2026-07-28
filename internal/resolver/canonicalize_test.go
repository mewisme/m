package resolver_test

import (
	"context"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/registry"
	"github.com/mewisme/mew/internal/resolver"
)

func dualWrapperPluginPackuments() map[string]registry.Packument {
	return map[string]registry.Packument{
		"peer-lib": {
			Name: "peer-lib", DistTags: map[string]string{"latest": "1.0.0"},
			Versions: map[string]registry.VersionMeta{
				"1.0.0": {Name: "peer-lib", Version: "1.0.0", Dist: registry.Dist{Integrity: "sha512-pl", Tarball: "peer-lib-1.0.0.tgz"}},
			},
		},
		"plugin-a": {
			Name: "plugin-a", DistTags: map[string]string{"latest": "1.0.0"},
			Versions: map[string]registry.VersionMeta{
				"1.0.0": {
					Name: "plugin-a", Version: "1.0.0",
					Dist:             registry.Dist{Integrity: "sha512-pa", Tarball: "plugin-a-1.0.0.tgz"},
					PeerDependencies: map[string]string{"peer-lib": "^1.0.0"},
				},
			},
		},
		"wrapper-a": {
			Name: "wrapper-a", DistTags: map[string]string{"latest": "1.0.0"},
			Versions: map[string]registry.VersionMeta{
				"1.0.0": {
					Name: "wrapper-a", Version: "1.0.0",
					Dist: registry.Dist{Integrity: "sha512-wa", Tarball: "wrapper-a-1.0.0.tgz"},
					Dependencies: map[string]string{
						"plugin-a": "1.0.0",
						"peer-lib": "1.0.0",
					},
				},
			},
		},
		"wrapper-b": {
			Name: "wrapper-b", DistTags: map[string]string{"latest": "1.0.0"},
			Versions: map[string]registry.VersionMeta{
				"1.0.0": {
					Name: "wrapper-b", Version: "1.0.0",
					Dist: registry.Dist{Integrity: "sha512-wb", Tarball: "wrapper-b-1.0.0.tgz"},
					Dependencies: map[string]string{
						"plugin-a": "1.0.0",
						"peer-lib": "1.0.0",
					},
				},
			},
		},
	}
}

func TestCanonicalizeMergesIdenticalPeerInstances(t *testing.T) {
	eng, _ := engineWithPackuments(t, dualWrapperPluginPackuments())
	root := writeProject(t, `{
  "name": "root",
  "version": "1.0.0",
  "dependencies": {
    "wrapper-a": "1.0.0",
    "wrapper-b": "1.0.0"
  }
}`)
	res, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{})
	if err != nil {
		t.Fatal(err)
	}
	var pluginKeys []string
	for _, p := range res.Graph.Packages {
		if p.ID.Name != "plugin-a" {
			continue
		}
		pluginKeys = append(pluginKeys, p.ID.Key())
	}
	if len(pluginKeys) != 1 {
		t.Fatalf("want 1 canonical plugin-a instance, got %d: %v", len(pluginKeys), pluginKeys)
	}
	if !strings.Contains(pluginKeys[0], "peer-lib@1.0.0") {
		t.Fatalf("plugin-a should carry peer-lib provider: %q", pluginKeys[0])
	}
}
