package envexec

import (
	"context"
	"path/filepath"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/binmeta"
)

// ProjectProvider plans and materializes local project importer environments.
type ProjectProvider struct{}

func (ProjectProvider) Kind() SourceKind { return SourceProject }

func (ProjectProvider) Plan(ctx context.Context, deps ProviderDeps, req ExecutionRequest) (EnvironmentPlan, error) {
	src, ok := req.Source.(ProjectSource)
	if !ok {
		return EnvironmentPlan{}, usageError("invalid project source")
	}
	imp, bind, linkerMode, err := resolveProjectImporter(ctx, deps, src.CWD, req.Filters)
	if err != nil {
		return EnvironmentPlan{}, err
	}
	id := EnvironmentIdentity{
		SchemaVersion:  IdentitySchemaVersion,
		Source:         SourceProject,
		GraphDigest:    bind.Fingerprint,
		MaterialDigest: bind.GenerationID,
		SourceDigest:   imp.RelativePath(),
		Platform:       CurrentPlatform(),
		LinkerMode:     linkerMode,
	}
	return EnvironmentPlan{
		SchemaVersion:      PlanSchemaVersion,
		Source:             SourceProject,
		Identity:           id,
		GraphDigest:        bind.Fingerprint,
		CacheState:         CacheCold,
		Materialization:    MaterializationNone,
		NetworkRequirement: NetworkRequirementNone,
		Verification:       req.Policy.Verification,
		SharedCache:        false,
		ReadOnly:           true,
		ProjectResolved: &ProjectResolvedPlan{
			ProjectRoot: imp.ProjectRoot(),
			PackageDir:  imp.PackageDir(),
			ImporterRel: imp.RelativePath(),
			Binding:     bind,
		},
	}, nil
}

func (ProjectProvider) Materialize(ctx context.Context, deps ProviderDeps, plan EnvironmentPlan) (PreparedEnvironment, error) {
	_ = ctx
	if plan.ProjectResolved == nil {
		return PreparedEnvironment{}, apperr.New(apperr.Internal, "envexec.project", "", "missing project plan data")
	}
	pr := plan.ProjectResolved
	nm := filepath.Join(pr.PackageDir, "node_modules")
	requireVerified := plan.Verification == VerificationRequired
	return PreparedEnvironment{
		SchemaVersion:    PreparedSchemaVersion,
		Source:           SourceProject,
		Identity:         plan.Identity,
		Root:             pr.ProjectRoot,
		ImporterRoot:     pr.PackageDir,
		ImporterRel:      pr.ImporterRel,
		NodeModules:      nm,
		WorkingDirectory: pr.PackageDir,
		Binding:          pr.Binding,
		RequireVerified:  requireVerified,
		AllowUnowned:     !requireVerified,
		ReadOnly:         true,
		SharedCache:      false,
		CacheState:       CacheCold,
		HostEnv:          hostEnvFrom(deps),
		LeasePolicy:      LeaseNone,
	}, nil
}

func resolveProjectImporter(ctx context.Context, deps ProviderDeps, cwd string, filters []string) (ImporterView, binmeta.GenerationBinding, string, error) {
	var empty binmeta.GenerationBinding
	if deps.DiscoverProject == nil || deps.SelectImporter == nil || deps.LoadBinding == nil {
		return nil, empty, "", apperr.New(apperr.Internal, "envexec.project", "", "missing project deps")
	}
	proj, err := deps.DiscoverProject(ctx, cwd)
	if err != nil {
		return nil, empty, "", err
	}
	imp, err := deps.SelectImporter(ctx, proj, filters)
	if err != nil {
		return nil, empty, "", err
	}
	bind, err := deps.LoadBinding(ctx, imp.ProjectRoot())
	if err != nil {
		return nil, empty, "", err
	}
	linkerMode := ""
	if deps.Config != nil {
		linkerMode = deps.Config.LinkerMode()
	}
	return imp, bind, linkerMode, nil
}
