package binresolve

import "github.com/mewisme/mew/internal/binmeta"

// LaunchKind describes how a child process should be started.
type LaunchKind string

const (
	LaunchDirect LaunchKind = "direct"
	LaunchCmd    LaunchKind = "cmd"
	LaunchNode   LaunchKind = "node"
)

// LaunchSpec is the platform-specific process launch plan.
type LaunchSpec struct {
	Program string
	Args    []string
	Dir     string
	Kind    LaunchKind
}

// Options configures bin resolution.
type Options struct {
	ProjectRoot     string
	ImporterRel     string // workspace-relative importer path ("." for root)
	PackageDir      string
	Command         string
	PackageFilter   string // --package dependency or alias name
	RequireVerified bool   // direct dispatch: OwnershipVerified required
	AllowUnowned    bool   // explicit m exec may use compatibility fallback
	GenerationID    string
	Fingerprint     string
	RequestCache    map[string]*binmeta.Document
}

// Result is the outcome of bin resolution.
type Result struct {
	Candidate    binmeta.BinCandidate
	Ambiguity    []binmeta.BinCandidate
	UsedFallback bool
	MissMessage  string
}
