package lockfile_test

import (
	"context"
	"testing"

	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/lockfile"
)

type fakeAdapter struct{}

func (fakeAdapter) Read(context.Context, string) (*graph.Graph, error) {
	return &graph.Graph{SchemaVersion: graph.SchemaVersion}, nil
}
func (fakeAdapter) Write(context.Context, string, *graph.Graph) error { return nil }

var _ lockfile.LockfileAdapter = fakeAdapter{}

func TestFakeLockfileAdapterSatisfiesInterface(t *testing.T) {
	var a lockfile.LockfileAdapter = fakeAdapter{}
	g, err := a.Read(context.Background(), "m.lock")
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Write(context.Background(), "m.lock", g); err != nil {
		t.Fatal(err)
	}
}
