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
	Field         string `json:"field"`
	Reason        string `json:"reason"`
	SourceFormat  string `json:"sourceFormat,omitempty"`
	Value         string `json:"value,omitempty"`
	SourcePath    string `json:"sourcePath,omitempty"`
	Semantic      bool   `json:"semantic,omitempty"`
	ProducerMajor int    `json:"producerMajor,omitempty"`
	Category      string `json:"category,omitempty"`
}

// LossReportSchemaVersion versions LossReport documents.
const LossReportSchemaVersion = 1

// LossReport collects round-trip fidelity gaps for lockfile adapters.
type LossReport struct {
	SchemaVersion int        `json:"schemaVersion"`
	Items         []LossItem `json:"items"`
	Extensions    Extensions `json:"extensions,omitempty"`
}

// DetectionConfidence classifies producer-major certainty.
type DetectionConfidence string

const (
	DetectionCertain  DetectionConfidence = "certain"
	DetectionInferred DetectionConfidence = "inferred"
)

// Detection records incumbent lock format and producer generation.
type Detection struct {
	Format        string // pnpm-v9 | pnpm-v10 | pnpm-v11 | nub
	ProducerMajor int    // 9–11 for pnpm; 0 for nub when not tied to a pnpm major
	Confidence    DetectionConfidence
	Evidence      []string
	ExplicitMajor bool // true when --pnpm-major disambiguates v9-shaped locks
}

// Certified reports whether detection is strong enough for incumbent encode/write.
func (d Detection) Certified() bool {
	if d.Format == "nub" {
		return true
	}
	if d.ExplicitMajor && d.ProducerMajor >= 9 && d.ProducerMajor <= 11 {
		return true
	}
	return d.Confidence == DetectionCertain
}

// WriteResult is the outcome of EncodePreserving. Adapters never touch live paths.
type WriteResult struct {
	Unchanged bool
	Bytes     []byte
}

// LockfileAdapter reads and writes lockfiles via the canonical graph.
type LockfileAdapter interface {
	Read(ctx context.Context, path string) (*graph.Graph, error)
	Write(ctx context.Context, path string, g *graph.Graph) error
}

// ExtensibleAdapter preserves format-specific extensions and supports byte-preserving writes.
type ExtensibleAdapter interface {
	LockfileAdapter
	ReadWithExtensions(ctx context.Context, path string) (*graph.Graph, Extensions, error)
	// WritePreserving returns nil without writing when the graph is unchanged (caller stages prior bytes).
	// Returns RepresentabilityError (+ LossReport) when mutation is lossy or unsupported.
	WritePreserving(ctx context.Context, path string, g *graph.Graph, prior []byte, ext Extensions, det Detection) error
	LossFromDocument(ctx context.Context, prior []byte) (LossReport, error)
}

// PreservingEncoder encodes incumbent locks without writing live paths (install txn staging).
type PreservingEncoder interface {
	EncodePreserving(ctx context.Context, path string, g *graph.Graph, prior []byte, ext Extensions, det Detection) (WriteResult, error)
}
