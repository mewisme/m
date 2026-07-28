package mlock_test

import (
	"testing"

	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/lockfile/mlock"
)

func TestMigrateV2AddsEdgeName(t *testing.T) {
	doc := &mlock.Document{
		LockfileVersion: 2,
		Settings:        mlock.DefaultSettings(),
		Importers: []mlock.ImporterSection{{
			ID: graph.RootImporter,
			Specifiers: []mlock.Specifier{
				{Name: "lodash", Range: "^4.17.0", Kind: graph.DepProd},
			},
		}},
		Packages: []graph.Package{{ID: graph.PackageID{Name: "lodash", Version: "4.17.21"}}},
		Edges: []graph.Edge{{
			From:  ".",
			To:    "lodash@4.17.21",
			Kind:  graph.DepProd,
			Range: "^4.17.0",
		}},
	}
	if err := mlock.Migrate(doc); err != nil {
		t.Fatal(err)
	}
	if doc.LockfileVersion != mlock.LockfileVersion {
		t.Fatalf("version=%d", doc.LockfileVersion)
	}
	if doc.Edges[0].Name != "lodash" {
		t.Fatalf("edge name=%q", doc.Edges[0].Name)
	}
}

func TestDecodeV2SkipsChecksum(t *testing.T) {
	raw := []byte(`{
  "lockfileVersion": 2,
  "checksum": "deadbeef",
  "settings": {
    "linker": "auto",
    "policy": {"schemaVersion": 1, "scriptTrust": "ask", "strictPeerDependencies": true}
  },
  "importers": [{
    "id": ".",
    "specifiers": [{"name": "lodash", "range": "^4.17.0", "kind": "prod"}]
  }],
  "packages": [{"id": {"name": "lodash", "version": "4.17.21"}}],
  "edges": [{"from": ".", "to": "lodash@4.17.21", "kind": "prod", "range": "^4.17.0"}]
}`)
	doc, err := mlock.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	if doc.LockfileVersion != 3 {
		t.Fatalf("version=%d", doc.LockfileVersion)
	}
	if doc.Edges[0].Name != "lodash" {
		t.Fatalf("name=%q", doc.Edges[0].Name)
	}
}
