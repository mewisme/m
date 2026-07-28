package resolver_test

import (
	"context"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/policy"
	"github.com/mewisme/mew/internal/registry"
	"github.com/mewisme/mew/internal/resolver"
)

func peerFixturePackuments() map[string]registry.Packument {
	return map[string]registry.Packument{
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
		"plugin-b": {
			Name: "plugin-b", DistTags: map[string]string{"latest": "1.0.0"},
			Versions: map[string]registry.VersionMeta{
				"1.0.0": {
					Name: "plugin-b", Version: "1.0.0",
					Dist:             registry.Dist{Integrity: "sha512-pb", Tarball: "plugin-b-1.0.0.tgz"},
					PeerDependencies: map[string]string{"peer-lib": "^2.0.0"},
				},
			},
		},
		"peer-lib": {
			Name: "peer-lib", DistTags: map[string]string{"latest": "2.0.0"},
			Versions: map[string]registry.VersionMeta{
				"1.0.0": {Name: "peer-lib", Version: "1.0.0", Dist: registry.Dist{Integrity: "sha512-pl1", Tarball: "peer-lib-1.0.0.tgz"}},
				"2.0.0": {Name: "peer-lib", Version: "2.0.0", Dist: registry.Dist{Integrity: "sha512-pl2", Tarball: "peer-lib-2.0.0.tgz"}},
			},
		},
		"needs-optional": {
			Name: "needs-optional", DistTags: map[string]string{"latest": "1.0.0"},
			Versions: map[string]registry.VersionMeta{
				"1.0.0": {
					Name: "needs-optional", Version: "1.0.0",
					Dist:             registry.Dist{Integrity: "sha512-no", Tarball: "needs-optional-1.0.0.tgz"},
					PeerDependencies: map[string]string{"missing-opt": "^1.0.0"},
					PeerDependenciesMeta: map[string]registry.PeerMeta{
						"missing-opt": {Optional: true},
					},
				},
			},
		},
		"@scope/host": {
			Name: "@scope/host", DistTags: map[string]string{"latest": "1.0.0"},
			Versions: map[string]registry.VersionMeta{
				"1.0.0": {
					Name: "@scope/host", Version: "1.0.0",
					Dist:             registry.Dist{Integrity: "sha512-sh", Tarball: "scope-host-1.0.0.tgz"},
					PeerDependencies: map[string]string{"@scope/peer": "^1.0.0"},
				},
			},
		},
		"@scope/peer": {
			Name: "@scope/peer", DistTags: map[string]string{"latest": "1.0.0"},
			Versions: map[string]registry.VersionMeta{
				"1.0.0": {Name: "@scope/peer", Version: "1.0.0", Dist: registry.Dist{Integrity: "sha512-sp", Tarball: "scope-peer-1.0.0.tgz"}},
			},
		},
		"multi-peer": {
			Name: "multi-peer", DistTags: map[string]string{"latest": "1.0.0"},
			Versions: map[string]registry.VersionMeta{
				"1.0.0": {
					Name: "multi-peer", Version: "1.0.0",
					Dist: registry.Dist{Integrity: "sha512-mp", Tarball: "multi-peer-1.0.0.tgz"},
					PeerDependencies: map[string]string{
						"peer-lib": "1.0.0",
						"react":    "^18.0.0",
					},
				},
			},
		},
		"react": {
			Name: "react", DistTags: map[string]string{"latest": "18.2.0"},
			Versions: map[string]registry.VersionMeta{
				"18.2.0": {Name: "react", Version: "18.2.0", Dist: registry.Dist{Integrity: "sha512-r", Tarball: "react-18.2.0.tgz"}},
			},
		},
		"wrapper": {
			Name: "wrapper", DistTags: map[string]string{"latest": "1.0.0"},
			Versions: map[string]registry.VersionMeta{
				"1.0.0": {
					Name: "wrapper", Version: "1.0.0",
					Dist:         registry.Dist{Integrity: "sha512-w", Tarball: "wrapper-1.0.0.tgz"},
					Dependencies: map[string]string{"plugin-a": "1.0.0"},
				},
			},
		},
	}
}

func TestPeerDualImporterProviders(t *testing.T) {
	eng, _ := engineWithPackuments(t, peerFixturePackuments())
	root := writeWorkspace(t, map[string]string{
		"package.json": `{
  "name": "root",
  "version": "1.0.0",
  "workspaces": ["packages/*"],
  "dependencies": { "peer-lib": "1.0.0" }
}`,
		"packages/a/package.json": `{
  "name": "pkg-a",
  "version": "1.0.0",
  "dependencies": { "plugin-a": "1.0.0", "peer-lib": "1.0.0" }
}`,
		"packages/b/package.json": `{
  "name": "pkg-b",
  "version": "1.0.0",
  "dependencies": { "plugin-b": "1.0.0", "peer-lib": "2.0.0" }
}`,
	})
	res, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{
		Policy: testStrictPeersOff(),
	})
	if err != nil {
		t.Fatal(err)
	}
	keys := map[string]string{}
	for _, p := range res.Graph.Packages {
		if p.ID.Name == "plugin-a" || p.ID.Name == "plugin-b" {
			keys[p.ID.Name] = p.ID.Key()
		}
	}
	if !strings.Contains(keys["plugin-a"], "peer-lib@1.0.0") {
		t.Fatalf("plugin-a should use importer-a peer provider: %q", keys["plugin-a"])
	}
	if !strings.Contains(keys["plugin-b"], "peer-lib@2.0.0") {
		t.Fatalf("plugin-b should use importer-b peer provider: %q", keys["plugin-b"])
	}
}

func TestPeerOptionalAbsent(t *testing.T) {
	eng, _ := engineWithPackuments(t, peerFixturePackuments())
	root := writeProject(t, `{
  "name": "root",
  "version": "1.0.0",
  "dependencies": { "needs-optional": "1.0.0" }
}`)
	res, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{
		Policy: &policy.Policy{StrictPeerDependencies: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range res.Graph.Packages {
		if p.ID.Name == "needs-optional" && len(p.ID.PeerProviderContext) != 0 {
			t.Fatalf("optional absent peer should not add providers: %#v", p.ID)
		}
	}
}

func TestPeerScopedProviders(t *testing.T) {
	eng, _ := engineWithPackuments(t, peerFixturePackuments())
	root := writeProject(t, `{
  "name": "root",
  "version": "1.0.0",
  "dependencies": {
    "@scope/host": "1.0.0",
    "@scope/peer": "1.0.0"
  }
}`)
	res, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range res.Graph.Packages {
		if p.ID.Name != "@scope/host" {
			continue
		}
		if len(p.ID.PeerProviderContext) != 1 || p.ID.PeerProviderContext[0].Name != "@scope/peer" {
			t.Fatalf("scoped peer providers=%#v", p.ID.PeerProviderContext)
		}
	}
}

func TestPeerAutoInstallAtImporter(t *testing.T) {
	eng, _ := engineWithPackuments(t, peerFixturePackuments())
	root := writeProject(t, `{
  "name": "root",
  "version": "1.0.0",
  "dependencies": { "wrapper": "1.0.0" }
}`)
	res, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{
		Policy: &policy.Policy{AutoInstallPeers: true, StrictPeerDependencies: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range res.Graph.Edges {
		if e.From == "wrapper@1.0.0" && strings.HasPrefix(e.To, "peer-lib@") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected auto-installed peer under wrapper: %#v", res.Graph.Edges)
	}
}
