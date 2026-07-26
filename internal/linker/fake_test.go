package linker_test

import (
	"context"
	"testing"

	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/linker"
)

type fakeLinker struct{}

func (fakeLinker) Plan(context.Context, *graph.Graph) (*linker.Plan, error) {
	return &linker.Plan{
		Ops: []linker.Op{{Kind: linker.OpMkdir, Dest: "node_modules"}},
	}, nil
}
func (fakeLinker) Apply(context.Context, *linker.Plan) error { return nil }

var _ linker.Linker = fakeLinker{}

func TestFakeLinkerSatisfiesInterface(t *testing.T) {
	var l linker.Linker = fakeLinker{}
	plan, err := l.Plan(context.Background(), &graph.Graph{SchemaVersion: graph.SchemaVersion})
	if err != nil {
		t.Fatal(err)
	}
	if err := l.Apply(context.Background(), plan); err != nil {
		t.Fatal(err)
	}
}
