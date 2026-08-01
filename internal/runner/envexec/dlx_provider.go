package envexec

import (
	"context"
	"os"
	"path/filepath"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/binmeta"
	"github.com/mewisme/mew/internal/runner/dlx"
)

// DLXProvider orchestrates mx ephemeral package execution.
type DLXProvider struct{}

func (DLXProvider) Kind() SourceKind { return SourceDLX }

func (p DLXProvider) Plan(ctx context.Context, deps ProviderDeps, req ExecutionRequest) (EnvironmentPlan, error) {
	src, ok := req.Source.(DLXSource)
	if !ok {
		return EnvironmentPlan{}, usageError("invalid dlx source")
	}
	if deps.DLX.TryLocal != nil && src.Mode == DLXModePackageCommand && len(src.Packages) > 0 && !src.Packages[0].HasExplicitVersion() {
		if env, ok, err := deps.DLX.TryLocal(ctx, req); err != nil {
			if apperr.CodeOf(err) != apperr.NotFound {
				return EnvironmentPlan{}, err
			}
		} else if ok {
			return EnvironmentPlan{
				SchemaVersion:      PlanSchemaVersion,
				Source:             SourceDLX,
				Identity:           env.Identity,
				GraphDigest:        env.Identity.GraphDigest,
				CacheState:         CacheWarm,
				Materialization:    MaterializationNone,
				NetworkRequirement: NetworkRequirementNone,
				Verification:       req.Policy.Verification,
				SharedCache:        false,
				ReadOnly:           true,
				Prepared:           &env,
			}, nil
		}
	}
	if deps.Config == nil {
		return EnvironmentPlan{}, apperr.New(apperr.Internal, "envexec.dlx", "", "missing config")
	}
	mxRoot := deps.Config.MXCacheDir()
	cacheRoot := deps.Config.CacheRoot()
	if src.Offline {
		return p.planOffline(ctx, deps, req, mxRoot, cacheRoot)
	}
	return p.planRemote(ctx, deps, req, mxRoot, cacheRoot)
}

func (p DLXProvider) planOffline(ctx context.Context, deps ProviderDeps, req ExecutionRequest, mxRoot, cacheRoot string) (EnvironmentPlan, error) {
	src := req.Source.(DLXSource)
	reqID := buildDLXRequestIdentity(deps, src)
	entry, err := dlx.LoadRequestIndex(dlx.RequestIndexPath(mxRoot, reqID.Digest()))
	if err != nil {
		return EnvironmentPlan{}, err
	}
	envDir := dlx.EnvironmentDir(mxRoot, entry.ResolvedEnvironmentDigest)
	identity := dlx.ResolvedEnvironmentIdentity{GraphDigest: entry.ResolvedEnvironmentDigest}
	warm := dlx.IsWarm(envDir)
	if warm {
		if err := dlx.VerifyWarmEnvironment(envDir, identity); err != nil {
			return EnvironmentPlan{}, err
		}
	}
	resolved := DLXResolveResult{Identity: identity}
	command, owner, err := selectDLXCommand(req, resolved)
	if err != nil {
		return EnvironmentPlan{}, err
	}
	consentKey := dlx.NewConsentKey(identity, command, owner)
	store, err := dlx.LoadConsentStore(dlx.ConsentStorePath(cacheRoot))
	if err != nil {
		return EnvironmentPlan{}, err
	}
	if !store.HasConsent(consentKey) && !src.Yes {
		return EnvironmentPlan{}, dlx.NonInteractiveUsageError()
	}
	id := dlxIdentityToEnv(identity)
	return EnvironmentPlan{
		SchemaVersion:      PlanSchemaVersion,
		Source:             SourceDLX,
		Identity:           id,
		GraphDigest:        identity.GraphDigest,
		CacheState:         cacheStateFromWarm(warm),
		Materialization:    materializationFromWarm(warm),
		NetworkRequirement: NetworkRequirementNone,
		Verification:       VerificationRequired,
		SharedCache:        true,
		ReadOnly:           true,
		SharedEnvDir:       envDir,
		DLXResolved: &DLXPlanData{
			Resolved:    resolved,
			Command:     command,
			Owner:       owner,
			EnvDir:      envDir,
			Warm:        warm,
			MXCacheRoot: mxRoot,
			CacheRoot:   cacheRoot,
			Offline:     true,
		},
		Request: req,
	}, nil
}

func emitPrep(deps ProviderDeps, label string) {
	if deps.PrepStage != nil {
		deps.PrepStage(label)
	}
}

func (p DLXProvider) planRemote(ctx context.Context, deps ProviderDeps, req ExecutionRequest, mxRoot, cacheRoot string) (EnvironmentPlan, error) {
	if deps.DLX.ResolveMetadata == nil {
		return EnvironmentPlan{}, apperr.New(apperr.Internal, "envexec.dlx", "", "missing resolve hook")
	}
	src := req.Source.(DLXSource)
	specLabel := "package"
	if len(src.Packages) > 0 {
		specLabel = src.Packages[0].Raw
		if specLabel == "" {
			specLabel = src.Packages[0].Name
		}
	}
	emitPrep(deps, "Resolving "+specLabel)
	resolved, err := deps.DLX.ResolveMetadata(ctx, req)
	if err != nil {
		return EnvironmentPlan{}, err
	}
	release, err := dlx.AcquireLock(ctx, mxRoot, dlx.LockRequest, resolved.RequestID.Digest())
	if err != nil {
		return EnvironmentPlan{}, err
	}
	if err := dlx.PublishRequestIndex(dlx.RequestIndexPath(mxRoot, resolved.RequestID.Digest()), dlx.RequestIndex{
		RequestDigest:              resolved.RequestID.Digest(),
		ResolvedEnvironmentDigest:  resolved.Identity.Digest(),
		NormalizedRequestedSpecs:   dlx.SortSpecs(req.Source.(DLXSource).Packages),
		ResolvedDirectPackageKeys:  resolved.DirectKeys,
		SanitizedRegistryOrigin:    resolved.Identity.SanitizedRegistryOrigin,
		TargetPlatformFingerprint:  resolved.Identity.TargetPlatformFingerprint,
		NodeFingerprint:            resolved.Identity.NodeFingerprint,
		ResolverPolicyFingerprint:  resolved.Identity.ResolverPolicyFingerprint,
		LifecyclePolicyFingerprint: resolved.Identity.LifecyclePolicyFingerprint,
		LinkerMode:                 resolved.Identity.LinkerMode,
		TransactionID:              resolved.TxnID,
	}); err != nil {
		release()
		return EnvironmentPlan{}, err
	}
	release()

	envDir := dlx.EnvironmentDir(mxRoot, resolved.Identity.Digest())
	warm := dlx.IsWarm(envDir)
	if warm {
		if err := dlx.VerifyWarmEnvironment(envDir, resolved.Identity); err != nil {
			if _, qerr := dlx.QuarantineEnvironment(mxRoot, envDir, resolved.Identity.Digest(), err.Error()); qerr == nil {
				warm = false
			} else {
				return EnvironmentPlan{}, err
			}
		}
	}
	command, owner, err := selectDLXCommand(req, resolved)
	if err != nil {
		return EnvironmentPlan{}, err
	}
	consentKey := dlx.NewConsentKey(resolved.Identity, command, owner)
	store, err := dlx.LoadConsentStore(dlx.ConsentStorePath(cacheRoot))
	if err != nil {
		return EnvironmentPlan{}, err
	}
	interactive := deps.DLX.Interactive != nil && deps.DLX.Interactive()
	decision := dlx.EvaluateConsent(warm, store, consentKey, src.Yes, interactive)
	if decision.NeedTTY {
		return EnvironmentPlan{}, dlx.NonInteractiveUsageError()
	}
	if !decision.Approved {
		if !warm {
			emitPrep(deps, "Checking consent")
		}
		if deps.Suspend != nil {
			_ = deps.Suspend(ctx)
		}
		if deps.Resume != nil {
			defer func() { _ = deps.Resume(ctx) }()
		}
		ok, perr := dlx.PromptConsent(ctx, deps.Prompter, resolved.Identity.Digest())
		if perr != nil {
			return EnvironmentPlan{}, perr
		}
		if !ok {
			return EnvironmentPlan{}, dlx.DeniedPolicyError()
		}
	}
	if err := dlx.MergeConsent(ctx, cacheRoot, mxRoot, consentKey); err != nil {
		return EnvironmentPlan{}, err
	}
	id := dlxIdentityToEnv(resolved.Identity)
	netReq := NetworkRequirementMetadata
	if warm {
		netReq = NetworkRequirementNone
	}
	return EnvironmentPlan{
		SchemaVersion:      PlanSchemaVersion,
		Source:             SourceDLX,
		Identity:           id,
		GraphDigest:        resolved.Identity.GraphDigest,
		CacheState:         cacheStateFromWarm(warm),
		Materialization:    materializationFromWarm(warm),
		NetworkRequirement: netReq,
		Verification:       VerificationRequired,
		SharedCache:        true,
		ReadOnly:           true,
		SharedEnvDir:       envDir,
		DLXResolved: &DLXPlanData{
			Resolved:    resolved,
			Command:     command,
			Owner:       owner,
			EnvDir:      envDir,
			Warm:        warm,
			MXCacheRoot: mxRoot,
			CacheRoot:   cacheRoot,
		},
		Request: req,
	}, nil
}

func (p DLXProvider) Materialize(ctx context.Context, deps ProviderDeps, plan EnvironmentPlan) (PreparedEnvironment, error) {
	if plan.DLXResolved == nil {
		return PreparedEnvironment{}, apperr.New(apperr.Internal, "envexec.dlx", "", "missing dlx plan data")
	}
	d := plan.DLXResolved
	envDir := d.EnvDir
	if !d.Warm {
		// Offline mode: refuse to build when network is forbidden.
		if d.Offline || plan.NetworkRequirement == NetworkRequirementNone {
			return PreparedEnvironment{}, apperr.New(apperr.Network, "envexec.dlx", d.Resolved.Identity.Digest(),
				"offline: environment not cached and network is forbidden")
		}
		envRelease, err := dlx.AcquireLock(ctx, d.MXCacheRoot, dlx.LockEnvironment, d.Resolved.Identity.Digest())
		if err != nil {
			return PreparedEnvironment{}, err
		}
		if !dlx.IsWarm(envDir) {
			if deps.DLX.BuildEnvironment == nil {
				envRelease()
				return PreparedEnvironment{}, apperr.New(apperr.Internal, "envexec.dlx", "", "missing build hook")
			}
			emitPrep(deps, "Fetching package")
			emitPrep(deps, "Preparing environment")
			if err := deps.DLX.BuildEnvironment(ctx, plan.Request, d.Resolved, envDir); err != nil {
				envRelease()
				return PreparedEnvironment{}, err
			}
		}
		envRelease()
	}
	if err := dlx.VerifyWarmEnvironment(envDir, d.Resolved.Identity); err != nil {
		return PreparedEnvironment{}, err
	}
	dlx.TouchAccess(d.MXCacheRoot, d.Resolved.Identity.Digest())
	bind, err := loadBindingFile(envDir)
	if err != nil {
		return PreparedEnvironment{}, err
	}
	return PreparedEnvironment{
		SchemaVersion:    PreparedSchemaVersion,
		Source:           SourceDLX,
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
		CommandOwner:     d.Owner,
	}, nil
}

func selectDLXCommand(req ExecutionRequest, resolved DLXResolveResult) (command, owner string, err error) {
	src := req.Source.(DLXSource)
	if src.Mode == DLXModePackageCommand {
		bins := dlx.BinNames(resolved.DirectBins[src.Packages[0].Name])
		command, err = dlx.InferModeABin(src.Packages[0].Name, bins)
		return command, src.Packages[0].Name, err
	}
	command = req.Command.Name
	owner, err = dlx.ResolveModeBCommand(command, resolved.DirectBins)
	return command, owner, err
}

func buildDLXRequestIdentity(deps ProviderDeps, src DLXSource) dlx.RequestIdentity {
	linkerMode := ""
	if deps.Config != nil {
		linkerMode = deps.Config.LinkerMode()
	}
	return dlx.RequestIdentity{
		NormalizedSpecs:           dlx.SortSpecs(src.Packages),
		TargetPlatformFingerprint: CurrentPlatform().OS + "/" + CurrentPlatform().Arch,
		LinkerMode:                linkerMode,
	}
}

func dlxIdentityToEnv(id dlx.ResolvedEnvironmentIdentity) EnvironmentIdentity {
	return EnvironmentIdentity{
		SchemaVersion:  IdentitySchemaVersion,
		Source:         SourceDLX,
		GraphDigest:    BareDigestHex(id.GraphDigest),
		MaterialDigest: id.Digest(),
		SourceDigest:   id.SanitizedRegistryOrigin,
		Platform:       CurrentPlatform(),
		LinkerMode:     id.LinkerMode,
	}
}

func cacheStateFromWarm(warm bool) CacheState {
	if warm {
		return CacheWarm
	}
	return CacheCold
}

func materializationFromWarm(warm bool) MaterializationRequirement {
	if warm {
		return MaterializationNone
	}
	return MaterializationRequired
}

func loadBindingFile(root string) (binmeta.GenerationBinding, error) {
	var empty binmeta.GenerationBinding
	path := filepath.Join(root, ".mew", "generation.json")
	b, err := os.ReadFile(path)
	if err != nil {
		return empty, apperr.Wrap(apperr.IO, "envexec.dlx", path, err)
	}
	return binmeta.DecodeGenerationBinding(b)
}

// DLXLeaseManager adapts dlx execution leases to envexec.LeaseManager.
type DLXLeaseManager struct {
	MXCacheRoot string
}

func (m DLXLeaseManager) Acquire(ctx context.Context, identity EnvironmentIdentity, holder string, pid int, token int64) (func(), error) {
	_ = ctx
	if m.MXCacheRoot == "" {
		return func() {}, nil
	}
	_, release, err := dlx.AcquireExecutionLease(m.MXCacheRoot, identity.MaterialDigest, holder, pid, token)
	if err != nil {
		return nil, err
	}
	return release, nil
}

var _ LeaseManager = DLXLeaseManager{}
