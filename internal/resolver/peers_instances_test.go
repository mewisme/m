package resolver_test

import (
	"context"
	"testing"

	"github.com/mewisme/mew/internal/registry"
	"github.com/mewisme/mew/internal/resolver"
)

func dualReactPluginPackuments() map[string]registry.Packument {
	return map[string]registry.Packument{
		"react": {
			Name: "react", DistTags: map[string]string{"latest": "19.0.0"},
			Versions: map[string]registry.VersionMeta{
				"18.2.0": {Name: "react", Version: "18.2.0", Dist: registry.Dist{Integrity: "sha512-r18", Tarball: "react-18.2.0.tgz"}},
				"19.0.0": {Name: "react", Version: "19.0.0", Dist: registry.Dist{Integrity: "sha512-r19", Tarball: "react-19.0.0.tgz"}},
			},
		},
		"plugin": {
			Name: "plugin", DistTags: map[string]string{"latest": "1.0.0"},
			Versions: map[string]registry.VersionMeta{
				"1.0.0": {
					Name: "plugin", Version: "1.0.0",
					Dist:             registry.Dist{Integrity: "sha512-pl", Tarball: "plugin-1.0.0.tgz"},
					PeerDependencies: map[string]string{"react": ">=18"},
				},
			},
		},
		"host-a": {
			Name: "host-a", DistTags: map[string]string{"latest": "1.0.0"},
			Versions: map[string]registry.VersionMeta{
				"1.0.0": {
					Name: "host-a", Version: "1.0.0",
					Dist:         registry.Dist{Integrity: "sha512-ha", Tarball: "host-a-1.0.0.tgz"},
					Dependencies: map[string]string{"react": "18.2.0", "plugin": "1.0.0"},
				},
			},
		},
		"host-b": {
			Name: "host-b", DistTags: map[string]string{"latest": "1.0.0"},
			Versions: map[string]registry.VersionMeta{
				"1.0.0": {
					Name: "host-b", Version: "1.0.0",
					Dist:         registry.Dist{Integrity: "sha512-hb", Tarball: "host-b-1.0.0.tgz"},
					Dependencies: map[string]string{"react": "19.0.0", "plugin": "1.0.0"},
				},
			},
		},
	}
}

func TestDistinctPluginInstancesPerReactContext(t *testing.T) {
	eng, _ := engineWithPackuments(t, dualReactPluginPackuments())
	root := writeProject(t, `{
  "name": "root",
  "version": "1.0.0",
  "dependencies": {
    "host-a": "1.0.0",
    "host-b": "1.0.0"
  }
}`)
	res, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{})
	if err != nil {
		t.Fatal(err)
	}
	var pluginKeys []string
	for _, p := range res.Graph.Packages {
		if p.ID.Name != "plugin" {
			continue
		}
		pluginKeys = append(pluginKeys, p.ID.Key())
	}
	if len(pluginKeys) != 2 {
		t.Fatalf("want 2 plugin instances, got %d: %v", len(pluginKeys), pluginKeys)
	}
	if pluginKeys[0] == pluginKeys[1] {
		t.Fatalf("plugin instances should differ by peer context: %v", pluginKeys)
	}
}
