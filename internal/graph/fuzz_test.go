package graph_test

import (
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/graph"
)

func FuzzDecodeGraph(f *testing.F) {
	f.Add([]byte(`{"schemaVersion":1,"importers":[{"id":"."}],"packages":[],"edges":[]}`))
	f.Add([]byte(`{`))
	f.Add([]byte(`{"schemaVersion":1,"importers":[],"packages":[{"id":{"name":"a","version":"1.0.0"}}],"edges":[{"from":".","to":"missing@1","kind":"prod"}]}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, err := graph.DecodeJSON(data)
		if err == nil {
			return
		}
		_ = apperr.CodeOf(err)
	})
}

func TestKnownBadGraph(t *testing.T) {
	_, err := graph.DecodeJSON([]byte(`{"schemaVersion":1,"importers":[{"id":"."}],"packages":[],"edges":[{"from":".","to":"x@1","kind":"prod"}]}`))
	if err == nil {
		t.Fatal("expected dangling edge error")
	}
	if apperr.CodeOf(err) != apperr.Lockfile {
		t.Fatalf("code %s", apperr.CodeOf(err))
	}
}
