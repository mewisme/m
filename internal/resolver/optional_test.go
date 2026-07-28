package resolver_test

import (
	"context"
	"testing"

	"github.com/mewisme/mew/internal/registry"
	"github.com/mewisme/mew/internal/resolver"
)

func optionalPackuments() map[string]registry.Packument {
	return map[string]registry.Packument{
		"root-only": {
			Name:     "root-only",
			DistTags: map[string]string{"latest": "1.0.0"},
			Versions: map[string]registry.VersionMeta{
				"1.0.0": {
					Name:    "root-only",
					Version: "1.0.0",
					Dist:    registry.Dist{Integrity: "sha512-root", Tarball: "root-only-1.0.0.tgz"},
				},
			},
		},
		"opt-darwin": {
			Name:     "opt-darwin",
			DistTags: map[string]string{"latest": "1.0.0"},
			Versions: map[string]registry.VersionMeta{
				"1.0.0": {
					Name:    "opt-darwin",
					Version: "1.0.0",
					Dist:    registry.Dist{Integrity: "sha512-opt", Tarball: "opt-darwin-1.0.0.tgz"},
					OS:      []string{"darwin"},
				},
			},
		},
		"opt-parent": {
			Name:     "opt-parent",
			DistTags: map[string]string{"latest": "1.0.0"},
			Versions: map[string]registry.VersionMeta{
				"1.0.0": {
					Name:    "opt-parent",
					Version: "1.0.0",
					Dist:    registry.Dist{Integrity: "sha512-parent", Tarball: "opt-parent-1.0.0.tgz"},
					OptionalDependencies: map[string]string{
						"missing-opt": "9.9.9",
					},
				},
			},
		},
	}
}

func TestResolveOptionalPlatformSkipped(t *testing.T) {
	if resolver.CurrentTarget().OS == "darwin" {
		t.Skip("host is darwin; platform skip not expected")
	}
	eng, _ := engineWithPackuments(t, optionalPackuments())
	root := writeProject(t, `{
  "name": "root",
  "version": "1.0.0",
  "optionalDependencies": { "opt-darwin": "1.0.0" },
  "dependencies": { "root-only": "1.0.0" }
}`)
	res, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range res.Graph.Packages {
		if p.ID.Name == "opt-darwin" {
			t.Fatalf("darwin-only optional should be skipped on this host: %#v", p)
		}
	}
	skipped := false
	for _, d := range res.Decisions {
		if d.Package == "opt-darwin" && d.Reason == "platform-skipped" {
			skipped = true
		}
	}
	if !skipped {
		t.Fatal("expected platform-skipped decision for opt-darwin")
	}
}

func TestResolveOptionalTransitiveFailure(t *testing.T) {
	eng, _ := engineWithPackuments(t, optionalPackuments())
	root := writeProject(t, `{
  "name": "root",
  "version": "1.0.0",
  "dependencies": { "opt-parent": "1.0.0" }
}`)
	res, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range res.Graph.Packages {
		if p.ID.Name == "missing-opt" {
			t.Fatal("optional transitive failure should not add package")
		}
	}
	found := false
	for _, d := range res.Decisions {
		if d.Package == "missing-opt" && d.Reason == "optional-failed" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected optional-failed decision for missing-opt")
	}
	_ = res
}
