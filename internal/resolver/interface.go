// Package resolver expands dependency graphs and records resolution decisions.
package resolver

import (
	"context"

	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/policy"
)

// ResolveOptions controls a resolve run.
type ResolveOptions struct {
	Policy *policy.Policy
	// Hints is an optional partial graph (e.g. prior lock) whose package
	// keys are preferred when the requested range still satisfies.
	Hints *graph.Graph
	// OmitRootDev skips root devDependencies when seeding the resolve queue
	// (e.g. m install --prod).
	OmitRootDev bool
}

// ResolutionDecision records candidate filtering and version selection for 0028.
type ResolutionDecision struct {
	Package     string            `json:"package"`
	Requested   string            `json:"requested"`
	Candidates  []string          `json:"candidates"`
	Selected    string            `json:"selected"`
	Reason      string            `json:"reason,omitempty"`
	Rejected    []string          `json:"rejected,omitempty"` // versions filtered by policy
	PeerContext graph.PeerContext `json:"peerContext,omitempty"`
}

// Resolution is an immutable resolve result: complete graph plus decision trace.
type Resolution struct {
	SchemaVersion int                  `json:"schemaVersion"`
	Graph         *graph.Graph         `json:"graph"`
	Decisions     []ResolutionDecision `json:"decisions"`
}

// ResolutionSchemaVersion versions serialized Resolution documents.
const ResolutionSchemaVersion = 1

// Resolver produces a complete immutable graph before any mutation.
type Resolver interface {
	Resolve(ctx context.Context, root string, opts ResolveOptions) (*Resolution, error)
}
