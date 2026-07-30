package envexec

import (
	"context"
	"path/filepath"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/fsx"
)

// CapsuleProvider executes from user-supplied capsule archives without registry contact.
type CapsuleProvider struct{}

func (CapsuleProvider) Kind() SourceKind { return SourceCapsule }

func (CapsuleProvider) Plan(ctx context.Context, deps ProviderDeps, req ExecutionRequest) (EnvironmentPlan, error) {
	src, ok := req.Source.(CapsuleSource)
	if !ok {
		return EnvironmentPlan{}, usageError("invalid capsule source")
	}
	if deps.CapsuleOpen == nil {
		return EnvironmentPlan{}, apperr.New(apperr.Internal, "envexec.capsule", "", "missing capsule opener")
	}
	open, err := deps.CapsuleOpen(ctx, src.Path)
	if err != nil {
		return EnvironmentPlan{}, err
	}
	linkerMode := ""
	if deps.Config != nil {
		linkerMode = deps.Config.LinkerMode()
	}
	id := EnvironmentIdentity{
		SchemaVersion:  IdentitySchemaVersion,
		Source:         SourceCapsule,
		GraphDigest:    open.ArchiveDigest,
		MaterialDigest: open.ArchiveDigest,
		SourceDigest:   src.Path,
		Platform:       CurrentPlatform(),
		LinkerMode:     linkerMode,
	}
	envDir := ""
	warm := false
	if deps.Config != nil {
		envDir = SharedCacheDir(deps.Config.CacheRoot(), SourceCapsule, id.IdentityDigest())
		warm = IsWarm(envDir)
		if warm {
			if err := VerifyWarmEnvironment(envDir, id); err != nil {
				warm = false
			}
		}
	}
	_ = open
	return EnvironmentPlan{
		SchemaVersion:      PlanSchemaVersion,
		Source:             SourceCapsule,
		Identity:           id,
		GraphDigest:        id.GraphDigest,
		CacheState:         cacheStateFromWarm(warm),
		Materialization:    materializationFromWarm(warm),
		NetworkRequirement: NetworkRequirementNone,
		Verification:       VerificationRequired,
		SharedCache:        true,
		ReadOnly:           true,
		SharedEnvDir:       envDir,
	}, nil
}

func (CapsuleProvider) Materialize(ctx context.Context, deps ProviderDeps, plan EnvironmentPlan) (PreparedEnvironment, error) {
	envDir := plan.SharedEnvDir
	if envDir == "" {
		return PreparedEnvironment{}, apperr.New(apperr.Internal, "envexec.capsule", "", "missing cache directory")
	}
	if !IsWarm(envDir) {
		if deps.Materializer == nil || deps.CapsuleOpen == nil {
			return PreparedEnvironment{}, apperr.New(apperr.Internal, "envexec.capsule", "", "missing materializer")
		}
		open, err := deps.CapsuleOpen(ctx, plan.Identity.SourceDigest)
		if err != nil {
			return PreparedEnvironment{}, err
		}
		txnID, err := fsx.NewLockID()
		if err != nil {
			return PreparedEnvironment{}, err
		}
		staging := StagingDir(filepath.Dir(envDir), plan.Identity.IdentityDigest(), txnID)
		spec := FrozenEnvironmentSpec{
			Identity:          plan.Identity,
			Graph:             open.Graph,
			LockSnapshot:      open.Lock,
			Manifest:          open.PackageJSON,
			LinkerMode:        plan.Identity.LinkerMode,
			NetworkPolicy:     NetworkForbidden,
			LifecyclePolicy:   LifecycleForbidden,
			SourceBlobDigest:  open.ArchiveDigest,
			PrelinkedExtracts: open.Blobs,
		}
		if err := deps.Materializer.Materialize(ctx, spec, staging); err != nil {
			return PreparedEnvironment{}, err
		}
		ready := ReadyMarker{
			SchemaVersion:  readySchemaVersion,
			IdentityDigest: plan.Identity.IdentityDigest(),
			GraphDigest:    plan.Identity.GraphDigest,
			Source:         string(SourceCapsule),
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
		Source:           SourceCapsule,
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
