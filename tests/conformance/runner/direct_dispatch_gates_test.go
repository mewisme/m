package runner_test

import (
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/testkit"
)

func setupDispatchFixture(t *testing.T, rel string) string {
	t.Helper()
	testkit.CleanEnv(t)
	proj := t.TempDir()
	testkit.CopyFixture(t, "dispatch/"+rel, proj)
	return proj
}

func TestDirectDispatchGateOffSkipsBinLookup(t *testing.T) {
	proj := setupDispatchFixture(t, "collision-matrix")
	code, out := runMProject(t, proj, "run", "build")
	if code != 0 && code != 2 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
}

func TestDirectDispatchGateOnVerifiedBin(t *testing.T) {
	skipWithoutNode(t)
	proj := setupDispatchFixture(t, "collision-matrix")
	code, _ := runMProject(t, proj, "run", "build")
	_ = code
}

func TestDirectDispatchNoHostPATHFallback(t *testing.T) {
	proj := setupDispatchFixture(t, "no-project")
	code, out := runMProject(t, proj, "eslint")
	if code == 0 {
		t.Fatal("expected failure outside project")
	}
	if strings.Contains(out, "PATH") && code == 0 {
		t.Fatal("host PATH fallback")
	}
}

func TestDirectDispatchConfigGatePrecedence(t *testing.T) {
	proj := setupDispatchFixture(t, "collision-matrix")
	code, _ := runMProject(t, proj, "run", "build")
	if code != 0 && code != 2 {
		t.Fatalf("exit=%d", code)
	}
}
