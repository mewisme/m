package runner_test

import (
	"context"
	"testing"

	"github.com/mewisme/mew/internal/runner/envexec"
)

func TestEnvexecCleanupIdempotent(t *testing.T) {
	calls := 0
	cleanup := envexec.CleanupFunc(func(context.Context) error {
		calls++
		return nil
	})
	_ = cleanup(context.Background())
	_ = cleanup(context.Background())
	if calls != 2 {
		t.Fatalf("calls=%d", calls)
	}
}

func TestEnvexecMaterializeFailureRollback(t *testing.T) {
	orch := &envexec.Orchestrator{Providers: envexec.DefaultProviderRegistry()}
	_, err := orch.Execute(context.Background(), envexec.ProviderDeps{}, envexec.ExecutionRequest{
		Source:  envexec.ProjectSource{CWD: "/nonexistent/project"},
		Command: envexec.CommandRequest{Name: "eslint"},
		Policy:  envexec.LockedProviderPolicy(envexec.SourceProject),
	})
	if err == nil {
		t.Fatal("expected failure")
	}
}
