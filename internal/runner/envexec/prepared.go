package envexec

import (
	"github.com/mewisme/mew/internal/binmeta"
)

// PreparedSchemaVersion is the current PreparedEnvironment schema version.
const PreparedSchemaVersion = 1

// PreparedEnvironment is a verified, launch-ready execution environment.
type PreparedEnvironment struct {
	SchemaVersion int
	Source        SourceKind
	Identity      EnvironmentIdentity

	Root             string
	ImporterRoot     string
	NodeModules      string
	WorkingDirectory string

	Binding         binmeta.GenerationBinding
	RequireVerified bool
	ReadOnly        bool
	SharedCache     bool
	CacheState      CacheState
	ImporterRel     string
	HostEnv         []string
	AllowUnowned    bool
	CommandOwner    string
	InferredCommand string

	CommandIndex binmeta.CommandIndex
	LeasePolicy  LeasePolicy
	Cleanup      CleanupFunc
}
