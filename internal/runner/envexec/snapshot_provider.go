package envexec

import (
	"context"
	"path/filepath"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/fsx"
)

// SnapshotPlanData carries snapshot record data between plan and materialize.
type SnapshotPlanData struct {
	ProjectRoot string
	Record      SnapshotLoadResult
}

// SnapshotProvider executes from project-local snapshot records without mutating the project.
type SnapshotProvider struct{}

func (SnapshotProvider) Kind() SourceKind { return SourceSnapshot }

func (SnapshotProvider) Plan(ctx context.Context, deps ProviderDeps, req ExecutionRequest) (EnvironmentPlan, error) {
	src, ok := req.Source.(SnapshotSource)
	if !ok {
		return EnvironmentPlan{}, usageError("invalid snapshot source")
	}
	if deps.SnapshotLoad == nil {
		return EnvironmentPlan{}, apperr.New(apperr.Internal, "envexec.snapshot", "", "missing snapshot loader")
	}
	rec, err := deps.SnapshotLoad(ctx, src.ProjectRoot, src.SnapshotID)
	if err != nil {
		return EnvironmentPlan{}, err
	}
	linkerMode := ""
	if deps.Config != nil {
		linkerMode = deps.Config.LinkerMode()
	}
	id := EnvironmentIdentity{
		SchemaVersion:  IdentitySchemaVersion,
		Source:         SourceSnapshot,
		GraphDigest:    rec.GraphDigest,
		MaterialDigest: rec.ID,
		SourceDigest:   src.SnapshotID,
		Platform:       CurrentPlatform(),
		LinkerMode:     linkerMode,
	}
	envDir := ""
	warm := false
	if deps.Config != nil {
		envDir = SharedCacheDir(deps.Config.CacheRoot(), SourceSnapshot, id.IdentityDigest())
		warm = IsWarm(envDir)
		if warm {
			if err := VerifyWarmEnvironment(envDir, id); err != nil {
				warm = false
			}
		}
	}
	plan := EnvironmentPlan{
		SchemaVersion:       PlanSchemaVersion,
		Source:              SourceSnapshot,
		Identity:            id,
		GraphDigest:         rec.GraphDigest,
		CacheState:          cacheStateFromWarm(warm),
		Materialization:     materializationFromWarm(warm),
		NetworkRequirement:  NetworkRequirementNone,
		Verification:        VerificationRequired,
		SharedCache:         true,
		ReadOnly:            true,
		SharedEnvDir:        envDir,
		SnapshotProjectRoot: src.ProjectRoot,
	}
	return plan, nil
}

func (SnapshotProvider) Materialize(ctx context.Context, deps ProviderDeps, plan EnvironmentPlan) (PreparedEnvironment, error) {
	envDir := plan.SharedEnvDir
	if envDir == "" {
		return PreparedEnvironment{}, apperr.New(apperr.Internal, "envexec.snapshot", "", "missing cache directory")
	}
	if !IsWarm(envDir) {
		if deps.Materializer == nil || deps.SnapshotLoad == nil || plan.SnapshotProjectRoot == "" {
			return PreparedEnvironment{}, apperr.New(apperr.Internal, "envexec.snapshot", "", "missing snapshot materialize deps")
		}
		rec, err := deps.SnapshotLoad(ctx, plan.SnapshotProjectRoot, plan.Identity.SourceDigest)
		if err != nil {
			return PreparedEnvironment{}, err
		}
		txnID, err := fsx.NewLockID()
		if err != nil {
			return PreparedEnvironment{}, err
		}
		staging := StagingDir(filepath.Dir(envDir), plan.Identity.IdentityDigest(), txnID)
		spec := FrozenEnvironmentSpec{
			Identity:        plan.Identity,
			Graph:           rec.Graph,
			LockSnapshot:    rec.Lock,
			Manifest:        rec.Manifest,
			MemberManifests: rec.MemberManifests,
			LinkerMode:      plan.Identity.LinkerMode,
			NetworkPolicy:   NetworkForbidden,
			LifecyclePolicy: LifecycleForbidden,
		}
		if err := deps.Materializer.Materialize(ctx, spec, staging); err != nil {
			return PreparedEnvironment{}, err
		}
		ready := ReadyMarker{
			SchemaVersion:  readySchemaVersion,
			IdentityDigest: plan.Identity.IdentityDigest(),
			GraphDigest:    plan.Identity.GraphDigest,
			Source:         string(SourceSnapshot),
			TargetPlatform: plan.Identity.Platform.OS + "/" + plan.Identity.Platform.Arch,
		}
		if err := PublishEnvironment(staging, envDir, ready); err != nil {
			return PreparedEnvironment{}, err
		}
	}
	if err := VerifyWarmEnvironment(envDir, plan.Identity); err != nil {
		return PreparedEnvironment{}, err
	}
	bind, err := loadBindingFile(envDir)
	if err != nil {
		return PreparedEnvironment{}, err
	}
	return PreparedEnvironment{
		SchemaVersion:    PreparedSchemaVersion,
		Source:           SourceSnapshot,
		Identity:         plan.Identity,
		Root:             envDir,
		ImporterRoot:     envDir,
		ImporterRel:      ".",
		NodeModules:      filepath.Join(envDir, "node_modules"),
		WorkingDirectory: envDir,
		Binding:          bind,
		RequireVerified:  true,
		AllowUnowned:     false,
		ReadOnly:         true,
		SharedCache:      true,
		CacheState:       CacheWarm,
		HostEnv:          hostEnvFrom(deps),
		LeasePolicy:      LeaseSharedCache,
	}, nil
}
