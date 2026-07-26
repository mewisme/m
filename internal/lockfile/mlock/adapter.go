package mlock

import (
	"context"
	"os"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/graph"
)

// Adapter implements lockfile.LockfileAdapter for native m.lock.
type Adapter struct{}

// Read decodes m.lock at path into a canonical graph.
func (Adapter) Read(_ context.Context, path string) (*graph.Graph, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, apperr.Wrap(apperr.IO, "mlock.read", path, err)
	}
	doc, err := Decode(data)
	if err != nil {
		return nil, err
	}
	return ToGraph(doc)
}

// Write encodes graph to m.lock at path (specifiers derived from importer edges).
func (Adapter) Write(_ context.Context, path string, g *graph.Graph) error {
	specs := SpecifiersFromGraph(g)
	doc, err := FromGraph(g, specs, DefaultSettings())
	if err != nil {
		return err
	}
	return WriteAtomic(path, doc)
}
