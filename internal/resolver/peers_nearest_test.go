package resolver_test

import (
	"context"
	"strings"
	"testing"

	"github.com/mewisme/m/internal/policy"
	"github.com/mewisme/m/internal/registry"
	"github.com/mewisme/m/internal/resolver"
)

func nearestPeerPackuments() map[string]registry.Packument {
	return map[string]registry.Packument{
		"consumer": {
			Name: "consumer", DistTags: map[string]string{"latest": "1.0.0"},
			Versions: map[string]registry.VersionMeta{
				"1.0.0": {
					Name: "consumer", Version: "1.0.0",
					Dist:             registry.Dist{Integrity: "sha512-c", Tarball: "consumer-1.0.0.tgz"},
					PeerDependencies: map[string]string{"peer-lib": "^1.0.0"},
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
		"parent": {
			Name: "parent", DistTags: map[string]string{"latest": "1.0.0"},
			Versions: map[string]registry.VersionMeta{
				"1.0.0": {
					Name: "parent", Version: "1.0.0",
					Dist:         registry.Dist{Integrity: "sha512-p", Tarball: "parent-1.0.0.tgz"},
					Dependencies: map[string]string{"peer-lib": "1.0.0", "consumer": "1.0.0"},
				},
			},
		},
		"root-peer": {
			Name: "root-peer", DistTags: map[string]string{"latest": "2.0.0"},
			Versions: map[string]registry.VersionMeta{
				"2.0.0": {Name: "root-peer", Version: "2.0.0", Dist: registry.Dist{Integrity: "sha512-rp", Tarball: "root-peer-2.0.0.tgz"}},
			},
		},
	}
}

func TestNearestIncompatiblePeerFailsStrict(t *testing.T) {
	packs := nearestPeerPackuments()
	packs["consumer"] = registry.Packument{
		Name: "consumer", DistTags: map[string]string{"latest": "1.0.0"},
		Versions: map[string]registry.VersionMeta{
			"1.0.0": {
				Name: "consumer", Version: "1.0.0",
				Dist:             registry.Dist{Integrity: "sha512-c", Tarball: "consumer-1.0.0.tgz"},
				PeerDependencies: map[string]string{"peer-lib": "^2.0.0"},
			},
		},
	}
	eng, _ := engineWithPackuments(t, packs)
	root := writeProject(t, `{
  "name": "root",
  "version": "1.0.0",
  "dependencies": {
    "root-peer": "2.0.0",
    "parent": "1.0.0"
  }
}`)
	_, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{
		Policy: &policy.Policy{StrictPeerDependencies: true},
	})
	if err == nil {
		t.Fatal("expected strict nearest-incompatible peer error")
	}
	if !strings.Contains(err.Error(), "incompatible peer") {
		t.Fatalf("want incompatible peer error: %v", err)
	}
}

func TestNearestCompatiblePeerWins(t *testing.T) {
	eng, _ := engineWithPackuments(t, nearestPeerPackuments())
	root := writeProject(t, `{
  "name": "root",
  "version": "1.0.0",
  "dependencies": {
    "peer-lib": "2.0.0",
    "parent": "1.0.0"
  }
}`)
	res, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{
		Policy: testStrictPeersOff(),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range res.Graph.Packages {
		if p.ID.Name != "consumer" {
			continue
		}
		if len(p.ID.PeerProviderContext) == 0 {
			t.Fatalf("consumer missing peer context: %#v", p.ID)
		}
		if p.ID.PeerProviderContext[0].Version != "1.0.0" {
			t.Fatalf("nearest compatible peer should be 1.0.0, got %#v", p.ID.PeerProviderContext)
		}
	}
}
