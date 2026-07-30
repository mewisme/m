package runner_test

import (
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/registry"
	"github.com/mewisme/mew/internal/testkit"
)

func TestCapsuleSecurityNoNetwork(t *testing.T) {
	testkit.CleanEnv(t)
	registry.TestProbeReset()
	proj := testkit.SetupExecFixture(t, "tool")
	code, _ := runMProjectArgv(t, proj, "env", "inspect", "capsule", filepath.Join(proj, "missing.mcap"))
	if code == 0 {
		t.Fatal("expected failure for missing capsule")
	}
	if registry.TestProbeMetadataCalls() != 0 || registry.TestProbeArtifactCalls() != 0 {
		t.Fatalf("network during capsule inspect")
	}
}

func TestCapsuleSecurityNoArtifactFetch(t *testing.T) {
	testkit.CleanEnv(t)
	registry.TestProbeReset()
	proj := testkit.SetupExecFixture(t, "tool")
	_, _ = runMProjectArgv(t, proj, "env", "inspect", "capsule", filepath.Join(proj, "tool.mcap"))
	if registry.TestProbeArtifactCalls() != 0 {
		t.Fatalf("artifact calls=%d", registry.TestProbeArtifactCalls())
	}
}
