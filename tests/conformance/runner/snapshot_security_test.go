package runner_test

import (
	"testing"

	"github.com/mewisme/mew/internal/registry"
	"github.com/mewisme/mew/internal/testkit"
)

func TestSnapshotSecurityNoNetwork(t *testing.T) {
	testkit.CleanEnv(t)
	registry.TestProbeReset()
	proj := testkit.SetupExecFixture(t, "tool")
	code, _ := runMProjectArgv(t, proj, "env", "inspect", "snapshot", "missing-id")
	if code == 0 {
		t.Fatal("expected failure for missing snapshot")
	}
	if registry.TestProbeMetadataCalls() != 0 || registry.TestProbeArtifactCalls() != 0 {
		t.Fatalf("network during snapshot inspect")
	}
}

func TestSnapshotSecurityNoArtifactFetch(t *testing.T) {
	testkit.CleanEnv(t)
	registry.TestProbeReset()
	proj := testkit.SetupExecFixture(t, "tool")
	_, _ = runMProjectArgv(t, proj, "env", "inspect", "snapshot", "snap-1")
	if registry.TestProbeArtifactCalls() != 0 {
		t.Fatalf("artifact calls=%d", registry.TestProbeArtifactCalls())
	}
}
