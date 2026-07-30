package envexec

import "context"

// EnvironmentProvider plans and materializes one source kind.
type EnvironmentProvider interface {
	Kind() SourceKind

	Plan(ctx context.Context, deps ProviderDeps, req ExecutionRequest) (EnvironmentPlan, error)

	Materialize(ctx context.Context, deps ProviderDeps, plan EnvironmentPlan) (PreparedEnvironment, error)
}

// stubProvider is the Phase 1 placeholder until real providers land.
type stubProvider struct {
	kind SourceKind
}

func (p stubProvider) Kind() SourceKind { return p.kind }

func (p stubProvider) Plan(ctx context.Context, deps ProviderDeps, req ExecutionRequest) (EnvironmentPlan, error) {
	return EnvironmentPlan{}, ErrUnimplemented
}

func (p stubProvider) Materialize(ctx context.Context, deps ProviderDeps, plan EnvironmentPlan) (PreparedEnvironment, error) {
	return PreparedEnvironment{}, ErrUnimplemented
}
