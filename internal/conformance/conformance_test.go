package conformance

import (
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/testkit"
)

func TestLoadCoreManifest(t *testing.T) {
	root := testkit.ModuleRoot(t)
	path := filepath.Join(root, "tests", "conformance", "core-matrix", "manifest.json")
	m, err := LoadManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	if m.SchemaVersion != SchemaVersion || m.Matrix != CoreMatrix {
		t.Fatalf("manifest=%+v", m)
	}
	if len(m.Suites) < 10 {
		t.Fatalf("expected many suites, got %d", len(m.Suites))
	}
}

func TestRunCoreDryRun(t *testing.T) {
	root := testkit.ModuleRoot(t)
	report, err := RunCore(t.Context(), RunOptions{RepoRoot: root, DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Passed || !report.DryRun || len(report.Suites) == 0 {
		t.Fatalf("report=%+v", report)
	}
	for _, s := range report.Suites {
		if s.Status != StatusPlanned {
			t.Fatalf("suite %s status=%s want planned", s.ID, s.Status)
		}
	}
}

func TestRunCoreFilterIdentity(t *testing.T) {
	root := testkit.ModuleRoot(t)
	report, err := RunCore(t.Context(), RunOptions{RepoRoot: root, Filter: "identity"})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Suites) != 1 {
		t.Fatalf("suites=%d want 1", len(report.Suites))
	}
	if report.Suites[0].ID != "lock-bridge-yarn-identity" {
		t.Fatalf("suite id=%s", report.Suites[0].ID)
	}
	if report.Suites[0].Status != StatusPassed {
		t.Fatalf("suite failed: %+v", report.Suites[0])
	}
}

func TestFilterSuitesPrefix(t *testing.T) {
	suites := []Suite{{ID: "lock-bridge-npm"}, {ID: "lock-bridge-yarn-identity"}}
	got := FilterSuites(suites, "lock-bridge-n")
	if len(got) != 1 || got[0].ID != "lock-bridge-npm" {
		t.Fatalf("got=%v", got)
	}
}
