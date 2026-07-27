package resolver_test

import (
	"context"
	"testing"

	"github.com/mewisme/m/internal/registry"
	"github.com/mewisme/m/internal/resolver"
)

func aliasPackuments() map[string]registry.Packument {
	return map[string]registry.Packument{
		"bar": {
			Name: "bar", DistTags: map[string]string{"latest": "1.0.0"},
			Versions: map[string]registry.VersionMeta{
				"1.0.0": {Name: "bar", Version: "1.0.0", Dist: registry.Dist{Integrity: "sha512-bar", Tarball: "bar-1.0.0.tgz"}},
			},
		},
	}
}

func TestResolveNpmAliasEdgeName(t *testing.T) {
	eng, _ := engineWithPackuments(t, aliasPackuments())
	root := writeProject(t, `{
  "name": "root",
  "version": "1.0.0",
  "dependencies": {
    "foo": "npm:bar@^1.0.0"
  }
}`)
	res, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{})
	if err != nil {
		t.Fatal(err)
	}
	var edgeFound bool
	for _, e := range res.Graph.Edges {
		if e.From != "." {
			continue
		}
		if e.Name == "foo" && e.To == "bar@1.0.0" && e.Range == "^1.0.0" {
			edgeFound = true
		}
	}
	if !edgeFound {
		t.Fatalf("missing alias edge: %#v", res.Graph.Edges)
	}
}
