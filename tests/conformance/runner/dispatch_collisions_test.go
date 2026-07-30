package runner_test

import (
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/testkit"
)

func copyDispatchFixture(t *testing.T, rel string) string {
	t.Helper()
	testkit.CleanEnv(t)
	proj := t.TempDir()
	testkit.CopyFixture(t, "dispatch/"+rel, proj)
	return proj
}

func TestDispatchBuiltinBeatsScript(t *testing.T) {
	proj := copyDispatchFixture(t, "collision-matrix")
	code, out := runMProject(t, proj, "install")
	if code != 0 {
		t.Fatalf("install exit=%d out=%s", code, out)
	}
	code, out = runMProject(t, proj, "version")
	if code != 0 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	if strings.Contains(out, "script-version") {
		t.Fatalf("builtin should beat script: %s", out)
	}
}

func TestDispatchAliasPrecedence(t *testing.T) {
	proj := copyDispatchFixture(t, "collision-matrix")
	code, _ := runMProject(t, proj, "i")
	if code != 0 && code != 2 {
		t.Fatalf("unexpected exit=%d", code)
	}
}

func TestDispatchMalformedManifest(t *testing.T) {
	proj := copyDispatchFixture(t, "malformed-manifest")
	code, out := runMProject(t, proj, "build")
	if code == 0 {
		t.Fatal("expected malformed manifest failure")
	}
	if !strings.Contains(out, "ERR_M_MANIFEST") && !strings.Contains(out, "ERR_M_USAGE") {
		t.Fatalf("out=%s", out)
	}
}
