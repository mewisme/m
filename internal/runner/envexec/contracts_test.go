package envexec_test

import (
	"testing"

	"github.com/mewisme/mew/internal/runner/dlx"
	"github.com/mewisme/mew/internal/runner/envexec"
)

func TestLockedProviderPolicies(t *testing.T) {
	project := envexec.LockedProviderPolicy(envexec.SourceProject)
	if project.Network != envexec.NetworkForbidden || project.Mutation != envexec.MutationProjectForbidden {
		t.Fatalf("project policy: %+v", project)
	}
	snapshot := envexec.LockedProviderPolicy(envexec.SourceSnapshot)
	if snapshot.Network != envexec.NetworkForbidden || snapshot.Verification != envexec.VerificationRequired {
		t.Fatalf("snapshot policy: %+v", snapshot)
	}
	if envexec.LockedLifecyclePolicy(envexec.SourceSnapshot) != envexec.LifecycleForbidden {
		t.Fatal("snapshot lifecycle must be forbidden")
	}
	if envexec.LockedLeasePolicy(envexec.SourceDLX) != envexec.LeaseSharedCache {
		t.Fatal("dlx lease must be shared-cache")
	}
}

func TestSourceKinds(t *testing.T) {
	cases := []struct {
		src  envexec.SourceRequest
		kind envexec.SourceKind
	}{
		{envexec.ProjectSource{CWD: "/p"}, envexec.SourceProject},
		{envexec.DLXSource{Packages: []dlx.PackageSpec{{Name: "eslint"}}}, envexec.SourceDLX},
		{envexec.SnapshotSource{SnapshotID: "snap"}, envexec.SourceSnapshot},
		{envexec.CapsuleSource{Path: "tool.mcap"}, envexec.SourceCapsule},
	}
	for _, tc := range cases {
		if tc.src.Kind() != tc.kind {
			t.Fatalf("%T kind = %q, want %q", tc.src, tc.src.Kind(), tc.kind)
		}
	}
}

func TestDefaultProviderRegistryCoversAllKinds(t *testing.T) {
	reg := envexec.DefaultProviderRegistry()
	for _, kind := range []envexec.SourceKind{
		envexec.SourceProject,
		envexec.SourceDLX,
		envexec.SourceSnapshot,
		envexec.SourceCapsule,
	} {
		if reg.Provider(kind) == nil {
			t.Fatalf("missing provider for %q", kind)
		}
	}
}

func TestOrchestratorRequiresProviderDeps(t *testing.T) {
	orch := envexec.Orchestrator{Providers: envexec.DefaultProviderRegistry()}
	_, err := orch.Execute(t.Context(), envexec.ProviderDeps{}, envexec.ExecutionRequest{
		Source: envexec.ProjectSource{CWD: "/tmp"},
		Command: envexec.CommandRequest{
			Name: "eslint",
		},
		Policy: envexec.LockedProviderPolicy(envexec.SourceProject),
	})
	if err == nil {
		t.Fatal("expected error without provider deps")
	}
}
