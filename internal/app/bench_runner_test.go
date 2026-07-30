package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunnerBenchCasesSmokeDefault(t *testing.T) {
	cases := runnerBenchCases(RunnerBenchOptions{Profile: RunnerBenchProfileSmoke})
	if len(cases) != 1 || cases[0].ID != "project-script" {
		t.Fatalf("smoke cases = %#v, want project-script only", cases)
	}
}

func TestRunnerBenchCasesFullProfile(t *testing.T) {
	cases := runnerBenchCases(RunnerBenchOptions{Profile: RunnerBenchProfileFull})
	if len(cases) != 2 {
		t.Fatalf("full profile len = %d, want 2", len(cases))
	}
}

func TestRunnerBenchCasesCaseID(t *testing.T) {
	cases := runnerBenchCases(RunnerBenchOptions{CaseID: "dlx-warm"})
	if len(cases) != 1 || cases[0].ID != "dlx-warm" {
		t.Fatalf("case override = %#v", cases)
	}
}

func TestCompareRunnerBaselineComparable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "baseline.json")
	if err := os.WriteFile(path, []byte(`{
  "schemaVersion": 1,
  "fixtureDigest": "x",
  "commandVersion": "runner-bench-v1",
  "environment": {"os":"linux","arch":"amd64","machineClass":"ci","goVersion":"go1.26","nodeVersion":"v22"},
  "recordedAt": "2026-07-31T00:00:00Z",
  "cases": []
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cmp, err := compareRunnerBaseline(path, RunnerBenchResult{})
	if err != nil {
		t.Fatal(err)
	}
	if cmp.Status != "comparable" {
		t.Fatalf("status = %q, want comparable", cmp.Status)
	}
}

func TestCompareRunnerBaselineNotComparable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "baseline.json")
	if err := os.WriteFile(path, []byte(`{"schemaVersion": 99}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cmp, err := compareRunnerBaseline(path, RunnerBenchResult{})
	if err != nil {
		t.Fatal(err)
	}
	if cmp.Status != "not-comparable" {
		t.Fatalf("status = %q, want not-comparable", cmp.Status)
	}
}

func TestBenchRunnerMutuallyExclusiveCaseProfile(t *testing.T) {
	_, err := BenchRunner(t.Context(), &Context{}, RunnerBenchOptions{
		Profile: RunnerBenchProfileFull,
		CaseID:  "dlx-warm",
	})
	if err == nil {
		t.Fatal("expected error for case + profile")
	}
}
