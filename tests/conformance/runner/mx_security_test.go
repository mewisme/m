package runner_test

import (
	"os"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/registry"
	"github.com/mewisme/mew/internal/testkit"
)

func withNonInteractiveStdin(t *testing.T, fn func()) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdin
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = old
		_ = r.Close()
		_ = w.Close()
	})
	_ = w.Close()
	fn()
}

func TestMXSecurityConsentDeniedNoArtifact(t *testing.T) {
	testkit.CleanEnv(t)
	proj, _, _ := testkit.SetupMXRegistryProject(t, `{"name":"mx-consent-stub","version":"1.0.0"}`)
	withNonInteractiveStdin(t, func() {
		code, out := runMXInProject(t, proj, "pkg-cli@1.0.0")
		if code == 0 {
			t.Fatalf("expected consent denial out=%s", out)
		}
		if registry.TestProbeArtifactCalls() != 0 {
			t.Fatalf("artifact calls=%d", registry.TestProbeArtifactCalls())
		}
	})
}

func TestMXSecurityNoArtifactBeforeConsent(t *testing.T) {
	testkit.CleanEnv(t)
	proj, _, _ := testkit.SetupMXRegistryProject(t, `{"name":"mx-consent-stub","version":"1.0.0"}`)
	withNonInteractiveStdin(t, func() {
		code, out := runMXInProject(t, proj, "pkg-cli@1.0.0")
		if registry.TestProbeArtifactCalls() != 0 {
			t.Fatalf("artifact calls=%d before consent", registry.TestProbeArtifactCalls())
		}
		if !strings.Contains(out, "ERR_M") && code == 0 {
			t.Fatalf("unexpected success: %s", out)
		}
	})
}

func TestMXSecurityLocalHitNoNetwork(t *testing.T) {
	skipWithoutNode(t)
	testkit.CleanEnv(t)
	proj := testkit.SetupMXLocalFixture(t, "echo-bin")
	code, out := runMXInProject(t, proj, "demo-pkg", "--", "hello")
	if code != 0 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	if registry.TestProbeMetadataCalls() != 0 {
		t.Fatalf("metadata calls=%d", registry.TestProbeMetadataCalls())
	}
	if registry.TestProbeArtifactCalls() != 0 {
		t.Fatalf("artifact calls=%d", registry.TestProbeArtifactCalls())
	}
}

func TestMXSecurityOfflineNoNetwork(t *testing.T) {
	testkit.CleanEnv(t)
	proj, _, _ := testkit.SetupMXRegistryProject(t, `{"name":"mx-offline","version":"1.0.0"}`)
	_, _ = runMXInProject(t, proj, "--offline", "pkg-cli@1.0.0")
	if registry.TestProbeArtifactCalls() != 0 {
		t.Fatalf("artifact calls=%d", registry.TestProbeArtifactCalls())
	}
}

func TestMXSecurityNonTTYNoYes(t *testing.T) {
	testkit.CleanEnv(t)
	proj, _, _ := testkit.SetupMXRegistryProject(t, `{"name":"mx-consent-stub","version":"1.0.0"}`)
	withNonInteractiveStdin(t, func() {
		code, out := runMXInProject(t, proj, "pkg-cli@1.0.0")
		if code != 2 {
			t.Fatalf("exit=%d out=%s", code, out)
		}
		if registry.TestProbeArtifactCalls() != 0 {
			t.Fatalf("artifact calls=%d", registry.TestProbeArtifactCalls())
		}
	})
}
