package mlock

import (
	"context"
	"os"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/graph"
)

// Adapter implements lockfile.LockfileAdapter for native m.lock.
type Adapter struct{}

// Read decodes m.lock at path into a canonical graph.
func (Adapter) Read(ctx context.Context, path string) (*graph.Graph, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
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
func (Adapter) Write(ctx context.Context, path string, g *graph.Graph) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	specs := SpecifiersFromGraph(g)
	doc, err := FromGraph(g, specs, DefaultSettings())
	if err != nil {
		return err
	}
	return WriteAtomic(path, doc)
}
