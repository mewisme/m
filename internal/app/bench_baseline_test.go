package app

import "testing"

func TestMatchInstallBaselineCase(t *testing.T) {
	t.Parallel()
	baseline := InstallBaseline{
		SchemaVersion: installBaselineSchemaVersion,
		Cases: []InstallBaselineCase{{
			Name:          "medium-graph-warm",
			OS:            "windows",
			Arch:          "amd64",
			GoVersion:     "go1.26.5",
			RunnerClass:   "local-windows",
			BenchmarkMode: "warm",
			TotalMsMedian: 169,
		}},
	}
	result := BenchResult{
		Case:        "medium-graph-warm",
		Mode:        "warm",
		GoVersion:   "go1.26.5",
		OS:          "windows",
		Arch:        "amd64",
		RunnerClass: "local-windows",
	}
	got, ok := MatchInstallBaselineCase(baseline, result)
	if !ok || got.TotalMsMedian != 169 {
		t.Fatalf("match=%v got=%+v", ok, got)
	}
	result.RunnerClass = "local-linux"
	if _, ok := MatchInstallBaselineCase(baseline, result); ok {
		t.Fatal("expected no match for different runner class")
	}
}
