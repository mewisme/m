// Package resolver expands dependency graphs and records resolution decisions.
package resolver

import (
	"context"

	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/lockfile"
	"github.com/mewisme/m/internal/manifest"
	"github.com/mewisme/m/internal/policy"
)

// PriorFingerprints captures lockfile settings fingerprints for incremental update comparison.
type PriorFingerprints struct {
	OverridesFingerprint      string
	ResolverPolicyFingerprint string
	TargetPlatformFingerprint string
}

// ResolveOptions controls a resolve run.
type ResolveOptions struct {
	Policy *policy.Policy
	// Hints is an optional partial graph (e.g. prior lock) whose package
	// keys are preferred when the requested range still satisfies.
	Hints *graph.Graph
	// Prior is the full prior lock graph for incremental resolve (0020).
	// When set with UpdateTargets, packages outside the update closure reuse
	// pinned versions when specifiers and overrides are unchanged.
	Prior *graph.Graph
	// UpdateTargets limits re-resolution to these packages and their prior
	// dependency subtrees. Empty means all direct manifest dependencies.
	UpdateTargets []string
	// PriorOverrides is the overrides map from when Prior was written.
	// When nil, overrides are treated as unchanged for pin reuse.
	PriorOverrides map[string]string
	// PriorFingerprints is the settings fingerprint snapshot from when Prior was written.
	PriorFingerprints *PriorFingerprints
	// IncrementalUpdate enables update-closure pin reuse (m update). Install passes
	// Prior+Hints without this flag for legacy hint selection.
	IncrementalUpdate bool
	// OmitRootDev skips root devDependencies when seeding the resolve queue
	// (e.g. m install --prod).
	OmitRootDev bool
	// Recursive seeds every workspace member as an importer (-r).
	Recursive bool
	// Filter limits seeded importers to --filter pattern matches.
	Filter []string
	// MemberManifests supplies in-memory member package.json docs (add --filter).
	MemberManifests map[string]*manifest.Document
}

// ResolutionDecision records candidate filtering and version selection for 0028.
type ResolutionDecision struct {
	Package       string                    `json:"package"`
	Requested     string                    `json:"requested"`
	Candidates    []string                  `json:"candidates"`
	Selected      string                    `json:"selected"`
	Reason        string                    `json:"reason,omitempty"`
	Rejected      []string                  `json:"rejected,omitempty"` // versions filtered by policy
	PeerProviders graph.PeerProviderContext `json:"peerProviders,omitempty"`
	OverrideFrom  string                    `json:"overrideFrom,omitempty"`
}

// Resolution is an immutable resolve result: complete graph plus decision trace.
type Resolution struct {
	SchemaVersion int                  `json:"schemaVersion"`
	Graph         *graph.Graph         `json:"graph"`
	Decisions     []ResolutionDecision `json:"decisions"`
	Extensions    lockfile.Extensions  `json:"extensions,omitempty"`
}

// ResolutionSchemaVersion versions serialized Resolution documents.
const ResolutionSchemaVersion = 1

// Resolver produces a complete immutable graph before any mutation.
type Resolver interface {
	Resolve(ctx context.Context, root string, opts ResolveOptions) (*Resolution, error)
}
