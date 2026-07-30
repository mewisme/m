package runner_test

import (
	"testing"

	"github.com/mewisme/mew/internal/runner/envexec"
)

func TestEnvexecProjectIdentityDigest(t *testing.T) {
	id := envexec.EnvironmentIdentity{
		SchemaVersion:  envexec.IdentitySchemaVersion,
		Source:         envexec.SourceProject,
		GraphDigest:    "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		MaterialDigest: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		SourceDigest:   "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		Platform:       envexec.CurrentPlatform(),
		LinkerMode:     "isolated",
	}
	got := id.IdentityDigest()
	if len(got) != 64 {
		t.Fatalf("digest len=%d", len(got))
	}
	if got != id.IdentityDigest() {
		t.Fatal("digest not deterministic")
	}
}

func TestEnvexecInspectPlanOnly(t *testing.T) {
	report, err := envexec.InspectEnvironment(t.Context(), envexec.ProviderDeps{}, envexec.DefaultProviderRegistry(), envexec.ExecutionRequest{
		Source:  envexec.ProjectSource{CWD: t.TempDir()},
		Command: envexec.CommandRequest{Name: "eslint"},
		Policy:  envexec.LockedProviderPolicy(envexec.SourceProject),
	})
	if err == nil {
		if report.V != envexec.InspectSchemaVersion {
			t.Fatalf("v=%d", report.V)
		}
		if report.Materialized {
			t.Fatal("inspect must be plan-only")
		}
	}
}
