package envexec

import (
	"context"
	"io"

	"github.com/mewisme/mew/internal/binmeta"
	"github.com/mewisme/mew/internal/process"
	"github.com/mewisme/mew/internal/runner"
	"github.com/mewisme/mew/internal/runner/dlx"
)

// ProviderDeps carries narrow provider dependencies. No app.Context or CLI types.
type ProviderDeps struct {
	Config           ConfigView
	HostEnv          func() []string
	DiscoverProject  func(ctx context.Context, cwd string) (ProjectView, error)
	SelectImporter   func(ctx context.Context, proj ProjectView, filters []string) (ImporterView, error)
	LoadBinding      func(ctx context.Context, root string) (binmeta.GenerationBinding, error)
	LoadCommandIndex func(ctx context.Context, nodeModules string) (binmeta.CommandIndex, error)
	ResolveBin       func(ctx context.Context, opts BinResolveOptions) (binmeta.BinCandidate, error)
	DiscoverNode     func(ctx context.Context) (NodeView, error)
	Store            StoreView
	Materializer     FrozenMaterializer
	SnapshotLoad     func(ctx context.Context, projectRoot, id string) (SnapshotLoadResult, error)
	CapsuleOpen      func(ctx context.Context, path string) (CapsuleOpenResult, error)
	DLX              DLXHooks
	Reporter         ReporterView
	Clock            func() int64
	FS               FSView
	Stdin            io.Reader
	Stdout           io.Writer
	Stderr           io.Writer
	// PrepStage emits human-only cold-path stage labels (Resolving, Consent, …).
	PrepStage func(label string)
}

// ConfigView is the effective configuration surface providers may read.
type ConfigView interface {
	CacheRoot() string
	MXCacheDir() string
	LinkerMode() string
}

// ProjectView is a discovered project root and metadata handle.
type ProjectView interface {
	Root() string
}

// ImporterView is the selected importer package context.
type ImporterView interface {
	ProjectRoot() string
	PackageDir() string
	RelativePath() string
}

// BinResolveOptions configures bin resolution for providers.
type BinResolveOptions struct {
	ProjectRoot     string
	ImporterRel     string
	PackageDir      string
	Command         string
	OwnerDependency string
	RequireVerified bool
	GenerationID    string
	Fingerprint     string
}

// NodeView describes the selected Node runtime.
type NodeView interface {
	ABI() string
	Executable() string
}

// StoreView is content-addressed blob access.
type StoreView interface {
	Has(ctx context.Context, digest string) (bool, error)
}

// FrozenMaterializer links immutable graphs into a shared cache directory.
type FrozenMaterializer interface {
	Materialize(ctx context.Context, spec FrozenEnvironmentSpec, finalDir string) error
}

// FrozenEnvironmentSpec describes an immutable environment to materialize.
type FrozenEnvironmentSpec struct {
	Identity          EnvironmentIdentity
	Graph             any // *graph.Graph from caller; typed as any to avoid graph import here
	LockSnapshot      []byte
	Manifest          []byte
	MemberManifests   map[string][]byte
	LinkerMode        string
	NetworkPolicy     NetworkPolicy
	LifecyclePolicy   LifecycleMaterializationPolicy
	SourceBlobDigest  string
	PrelinkedExtracts map[string]string
}

// SnapshotLoadResult is a validated snapshot record for execution.
type SnapshotLoadResult struct {
	ID              string
	GraphDigest     string
	Lock            []byte
	Manifest        []byte
	MemberManifests map[string][]byte
	Graph           any
}

// CapsuleOpenResult is a validated capsule opened for execution.
type CapsuleOpenResult struct {
	Path          string
	ArchiveDigest string
	Manifest      any // *capsule.Manifest
	Graph         any
	Lock          []byte
	PackageJSON   []byte
	Blobs         map[string]string // algo/hex -> extract dir
}

// DLXResolveResult is resolved DLX metadata from the app layer.
type DLXResolveResult struct {
	Identity   dlx.ResolvedEnvironmentIdentity
	RequestID  dlx.RequestIdentity
	DirectKeys []string
	DirectBins map[string]map[string]string
	TxnID      string
	AppState   any
}

// DLXHooks exposes app-injected DLX resolve/build/local primitives.
type DLXHooks struct {
	ResolveMetadata  func(ctx context.Context, req ExecutionRequest) (DLXResolveResult, error)
	BuildEnvironment func(ctx context.Context, req ExecutionRequest, resolved DLXResolveResult, envDir string) error
	TryLocal         func(ctx context.Context, req ExecutionRequest) (PreparedEnvironment, bool, error)
	Interactive      func() bool
}

// ReporterView emits diagnostics without CLI coupling.
type ReporterView interface {
	Debug(msg string)
}

// FSView is a narrow filesystem abstraction for providers.
type FSView interface {
	Stat(path string) (bool, error)
}

// ExecAdapter launches a prepared environment through runner.Exec.
type ExecAdapter interface {
	Exec(ctx context.Context, env PreparedEnvironment, req ExecutionRequest, sup process.ProcessSupervisor) (runner.ExecResult, error)
}

// DefaultExecAdapter delegates to runner.Exec using PreparedEnvironment paths.
type DefaultExecAdapter struct{}

func (DefaultExecAdapter) Exec(ctx context.Context, env PreparedEnvironment, req ExecutionRequest, sup process.ProcessSupervisor) (runner.ExecResult, error) {
	hostEnv := env.HostEnv
	if len(hostEnv) == 0 {
		hostEnv = osEnviron()
	}
	cmd := req.Command.Name
	if cmd == "" {
		cmd = env.InferredCommand
	}
	return runner.Exec(ctx, runner.ExecOptions{
		ProjectRoot:     env.Root,
		PackageDir:      env.ImporterRoot,
		ImporterRel:     env.ImporterRel,
		NodeModules:     env.NodeModules,
		Command:         cmd,
		PackageFilter:   req.Command.OwnerDependency,
		ForwardedArgs:   req.Command.Args,
		HostEnv:         hostEnv,
		RequireVerified: env.RequireVerified,
		AllowUnowned:    env.AllowUnowned,
		GenerationID:    env.Binding.GenerationID,
		Fingerprint:     env.Binding.Fingerprint,
		Stdin:           req.IO.Stdin,
		Stdout:          req.IO.Stdout,
		Stderr:          req.IO.Stderr,
		Suspend:         req.Suspend,
		Resume:          req.Resume,
	}, sup)
}
