// Package lockfile owns format adapters over the canonical graph.
package lockfile

import (
	"context"
	"encoding/json"

	"github.com/mewisme/mew/internal/graph"
)

// Graph is the canonical dependency graph (alias for shared ownership docs).
type Graph = graph.Graph

// Extensions holds adapter-owned format-specific fields that must not leak into
// core graph algorithms.
type Extensions map[string]json.RawMessage

// LossItem records data that cannot be represented in a target lockfile format.
type LossItem struct {
	Field        string `json:"field"`
	Reason       string `json:"reason"`
	SourceFormat string `json:"sourceFormat,omitempty"`
	Value        string `json:"value,omitempty"`
}

// LossReportSchemaVersion versions LossReport documents.
const LossReportSchemaVersion = 1

// LossReport collects round-trip fidelity gaps for lockfile adapters.
type LossReport struct {
	SchemaVersion int        `json:"schemaVersion"`
	Items         []LossItem `json:"items"`
	Extensions    Extensions `json:"extensions,omitempty"`
}

// LockfileAdapter reads and writes lockfiles via the canonical graph.
type LockfileAdapter interface {
	Read(ctx context.Context, path string) (*graph.Graph, error)
	Write(ctx context.Context, path string, g *graph.Graph) error
}
