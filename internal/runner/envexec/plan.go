package envexec

import "github.com/mewisme/mew/internal/binmeta"

// PlanSchemaVersion is the current EnvironmentPlan JSON schema version.
const PlanSchemaVersion = 1

// EnvironmentPlan describes a resolved execution environment without mutating
// the user project or acquiring execution leases.
type EnvironmentPlan struct {
	SchemaVersion      int
	Source             SourceKind
	Identity           EnvironmentIdentity
	GraphDigest        string
	CacheState         CacheState
	Materialization    MaterializationRequirement
	NetworkRequirement NetworkRequirement
	Verification       VerificationPolicy
	SharedCache        bool
	ReadOnly           bool
	AvailableCommands  []CommandRecord
	Warnings           []Diagnostic

	// ProjectResolved is populated by ProjectProvider.Plan for materialization.
	ProjectResolved *ProjectResolvedPlan
	// SnapshotProjectRoot locates project-local snapshots during materialize.
	SnapshotProjectRoot string
	// SharedEnvDir is the warm shared cache directory when applicable.
	SharedEnvDir string
	// DLXResolved carries DLX-specific plan data.
	DLXResolved *DLXPlanData
	// Prepared skips materialize when already resolved (e.g. DLX local-first).
	Prepared *PreparedEnvironment
	// Request preserves the original execution request for materialize hooks.
	Request ExecutionRequest
}

// ProjectResolvedPlan holds importer paths resolved during planning.
type ProjectResolvedPlan struct {
	ProjectRoot string
	PackageDir  string
	ImporterRel string
	Binding     binmeta.GenerationBinding
}

// DLXPlanData carries DLX orchestration state between plan and materialize.
type DLXPlanData struct {
	Resolved    DLXResolveResult
	Command     string
	Owner       string
	EnvDir      string
	Warm        bool
	MXCacheRoot string
	CacheRoot   string
	Offline     bool
}

// CacheState reports environment cache classification for inspect output.
type CacheState string

const (
	CacheProject   CacheState = "project"
	CacheWarm      CacheState = "warm"
	CacheCold      CacheState = "cold"
	CacheEphemeral CacheState = "ephemeral"
)

// MaterializationRequirement describes whether materialization is needed.
type MaterializationRequirement string

const (
	MaterializationNone     MaterializationRequirement = "none"
	MaterializationRequired MaterializationRequirement = "required"
)

// NetworkRequirement summarizes network needs for the plan.
type NetworkRequirement string

const (
	NetworkRequirementNone      NetworkRequirement = "none"
	NetworkRequirementMetadata  NetworkRequirement = "metadata"
	NetworkRequirementArtifacts NetworkRequirement = "artifacts"
)

// CommandRecord is one command available in the planned environment.
type CommandRecord struct {
	Name  string `json:"name"`
	Owner string `json:"owner,omitempty"`
}

// Diagnostic is a deterministic plan warning or note.
type Diagnostic struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
