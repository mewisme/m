package envexec

import "context"

// EnvironmentProvider plans and materializes one source kind.
type EnvironmentProvider interface {
	Kind() SourceKind

	Plan(ctx context.Context, deps ProviderDeps, req ExecutionRequest) (EnvironmentPlan, error)

	Materialize(ctx context.Context, deps ProviderDeps, plan EnvironmentPlan) (PreparedEnvironment, error)
}
