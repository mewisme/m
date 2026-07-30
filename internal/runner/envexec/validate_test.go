package envexec_test

import (
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/runner/dlx"
	"github.com/mewisme/mew/internal/runner/envexec"
)

func TestValidateRequestMissingSource(t *testing.T) {
	err := envexec.ValidateRequest(envexec.ExecutionRequest{
		Command: envexec.CommandRequest{Name: "eslint"},
	})
	assertUsage(t, err)
}

func TestValidateRequestMissingCommand(t *testing.T) {
	err := envexec.ValidateRequest(envexec.ExecutionRequest{
		Source: envexec.ProjectSource{CWD: "/p"},
	})
	assertUsage(t, err)
}

func TestValidateSnapshotRejectsOwnerDependency(t *testing.T) {
	err := envexec.ValidateRequest(envexec.ExecutionRequest{
		Source: envexec.SnapshotSource{ProjectRoot: "/p", SnapshotID: "snap1"},
		Command: envexec.CommandRequest{
			Name:            "eslint",
			OwnerDependency: "eslint",
		},
	})
	assertUsage(t, err)
}

func TestValidateCapsuleRejectsOwnerDependency(t *testing.T) {
	err := envexec.ValidateRequest(envexec.ExecutionRequest{
		Source: envexec.CapsuleSource{Path: "tool.mcap"},
		Command: envexec.CommandRequest{
			Name:              "tool",
			OwnerDependency:   "tool",
			RequireOwnerMatch: true,
		},
	})
	assertUsage(t, err)
}

func TestValidateSnapshotRequiresProjectRoot(t *testing.T) {
	err := envexec.ValidateRequest(envexec.ExecutionRequest{
		Source:  envexec.SnapshotSource{SnapshotID: "snap1"},
		Command: envexec.CommandRequest{Name: "eslint"},
	})
	assertUsage(t, err)
}

func TestValidateRecursiveOnlyProject(t *testing.T) {
	err := envexec.ValidateRequest(envexec.ExecutionRequest{
		Source:    envexec.SnapshotSource{ProjectRoot: "/p", SnapshotID: "snap1"},
		Command:   envexec.CommandRequest{Name: "eslint"},
		Recursive: true,
	})
	assertUsage(t, err)
}

func TestValidateDLXRejectsOwnerMatch(t *testing.T) {
	err := envexec.ValidateRequest(envexec.ExecutionRequest{
		Source: envexec.DLXSource{Packages: []dlx.PackageSpec{{Name: "eslint"}}},
		Command: envexec.CommandRequest{
			Name:              "eslint",
			OwnerDependency:   "eslint",
			RequireOwnerMatch: true,
		},
	})
	assertUsage(t, err)
}

func TestValidateProjectAcceptsValidRequest(t *testing.T) {
	err := envexec.ValidateRequest(envexec.ExecutionRequest{
		Source:  envexec.ProjectSource{CWD: "/p"},
		Command: envexec.CommandRequest{Name: "eslint"},
		Policy:  envexec.LockedProviderPolicy(envexec.SourceProject),
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidatePolicyRejectsUnknownNetwork(t *testing.T) {
	err := envexec.ValidatePolicy(envexec.ExecutionPolicy{Network: "bogus"})
	assertUsage(t, err)
}

func assertUsage(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected usage error")
	}
	if apperr.CodeOf(err) != apperr.Usage {
		t.Fatalf("code = %q, want %q: %v", apperr.CodeOf(err), apperr.Usage, err)
	}
}
