package conformance

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/mewisme/mew/internal/testkit"
)

func TestLoadRunnerManifest(t *testing.T) {
	root := testkit.ModuleRoot(t)
	m, err := LoadRunnerManifest(RunnerManifestPath(root))
	if err != nil {
		t.Fatal(err)
	}
	if m.Matrix != RunnerMatrix || len(m.Suites) < 20 {
		t.Fatalf("manifest=%+v", m)
	}
	for _, suite := range m.Suites {
		if !runnerSuiteSupportedOnPlatform(suite, runtime.GOOS) {
			continue
		}
		if err := ValidateExpectedTestsRegex(root, suite); err != nil {
			t.Fatalf("suite %s: %v", suite.ID, err)
		}
	}
}

func TestRunnerManifestDigestsStable(t *testing.T) {
	root := testkit.ModuleRoot(t)
	m, err := LoadRunnerManifest(RunnerManifestPath(root))
	if err != nil {
		t.Fatal(err)
	}
	w, err := LoadWaiverManifest(RunnerWaiverPath(root))
	if err != nil {
		t.Fatal(err)
	}
	md, err := RunnerManifestDigest(m)
	if err != nil || len(md) != 64 {
		t.Fatalf("manifest digest=%q err=%v", md, err)
	}
	wd, err := RunnerWaiverManifestDigest(w)
	if err != nil || len(wd) != 64 {
		t.Fatalf("waiver digest=%q err=%v", wd, err)
	}
}

func TestSelectRunnerSuitesIntersection(t *testing.T) {
	root := testkit.ModuleRoot(t)
	m, err := LoadRunnerManifest(RunnerManifestPath(root))
	if err != nil {
		t.Fatal(err)
	}
	selected, err := SelectRunnerSuites(m.Suites, []string{"dispatch"}, []string{"runner-dispatch-collisions"})
	if err != nil {
		t.Fatal(err)
	}
	if len(selected) != 1 || selected[0].ID != "runner-dispatch-collisions" {
		t.Fatalf("selected=%v", selected)
	}
}

func TestValidateRunRegexAnchored(t *testing.T) {
	if err := validateRunRegex("TestFoo"); err == nil {
		t.Fatal("expected anchored regex error")
	}
	if err := validateRunRegex(`^(TestA\|TestB)$`); err == nil {
		t.Fatal("expected escaped pipe error")
	}
	if err := validateRunRegex("^(TestA|TestB)$"); err != nil {
		t.Fatal(err)
	}
}

func TestRunnerManifestPath(t *testing.T) {
	root := testkit.ModuleRoot(t)
	want := filepath.Join(root, "tests", "conformance", "runner-matrix", "manifest.json")
	if RunnerManifestPath(root) != want {
		t.Fatalf("got %s", RunnerManifestPath(root))
	}
}
