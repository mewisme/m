package resolver_test

import (
	"context"
	"testing"

	"github.com/mewisme/m/internal/resolver"
)

func TestResolveCatalogDep(t *testing.T) {
	eng, _ := testEngine(t)
	root := writeWorkspace(t, map[string]string{
		"package.json": `{
  "name":"root",
  "catalog": {"lodash":"4.17.21"},
  "dependencies": {"lodash":"catalog:"}
}`,
	})
	res, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{
		Policy: testStrictPeersOff(),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range res.Graph.Packages {
		if p.ID.Name == "lodash" && p.ID.Version == "4.17.21" {
			return
		}
	}
	t.Fatalf("lodash not resolved: %#v", res.Graph.Packages)
}

func TestResolveCatalogMissingEntry(t *testing.T) {
	eng, _ := testEngine(t)
	root := writeWorkspace(t, map[string]string{
		"package.json": `{
  "name":"root",
  "dependencies": {"lodash":"catalog:"}
}`,
	})
	_, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{})
	if err == nil {
		t.Fatal("expected error")
	}
}
